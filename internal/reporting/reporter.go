// Package reporting is the event bus between the run loop and whatever is
// watching it. All() and the processors do not know a TUI exists: they emit
// events; a TUIReporter forwards them to the program, and LogReporter
// reproduces the pre-event streaming output exactly (the byte-identical
// headless contract).
package reporting

import (
	"os"
	"os/exec"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// Reporter consumes run events.
type Reporter interface {
	Emit(Event)
}

// Event is one run occurrence. The concrete set below is closed by design:
// the display layer switches over it.
type Event interface{ runEvent() }

// ProcStarted marks a processor beginning its files.
type ProcStarted struct {
	Processor string
	Files     int
	Providers []string
}

// ProcFinished marks a processor done, with its error if it failed.
type ProcFinished struct {
	Processor string
	Err       error
	Dur       time.Duration
}

// ProcSkipped marks a processor that did not run.
type ProcSkipped struct {
	Processor string
	Reason    string
}

// LaneUpdate is per-provider progress inside a processor.
type LaneUpdate struct {
	Processor string
	Provider  string
	Done      int
	Total     int
	Status    types.Status
}

// ResourceDone reports one completed unit of work.
type ResourceDone struct {
	Resource types.Resource
}

// TerminalReq asks the display layer to hand the real terminal to a command.
// stderr of interactive commands is never piped - capturing it swallows
// sudo's password prompt and hangs the run.
type TerminalReq struct {
	Processor string
	Cmd       *exec.Cmd
	Done      chan error
}

// TerminalFunc asks the display layer to lend the real terminal to an
// in-process interaction - a huh form, a raw stdin prompt. The TerminalReq
// counterpart for code that runs in this process rather than a child.
type TerminalFunc struct {
	Run  func() error
	Done chan error
}

// HaltDecision is the operator's answer to an interactive halt.
type HaltDecision int

const (
	HaltAbort HaltDecision = iota
	HaltRetry
	HaltSkip
)

// HaltReq reports a processor error in an interactive run and waits for the
// operator: retry the processor, skip past the error, or abort the run. The
// LogReporter answers abort, which is exactly the pre-halt behavior (an
// interactive headless run returned the error immediately).
type HaltReq struct {
	Processor string
	Err       error
	Decision  chan HaltDecision
}

// RunFinished carries the collected step errors of a push-through run.
type RunFinished struct {
	Errs []types.StepError
}

func (ProcStarted) runEvent()  {}
func (ProcFinished) runEvent() {}
func (ProcSkipped) runEvent()  {}
func (LaneUpdate) runEvent()   {}
func (ResourceDone) runEvent() {}
func (TerminalReq) runEvent()  {}
func (TerminalFunc) runEvent() {}
func (HaltReq) runEvent()      {}
func (RunFinished) runEvent()  {}

// processorLabels are the exact strings the pre-event loop logged per
// processor; LogReporter must reproduce them byte-for-byte.
var processorLabels = map[string]string{
	types.BlueprintTypeRepositories:  "Processing repositories",
	types.BlueprintTypePackages:      "Processing packages",
	types.BlueprintTypeFiles:         "Processing files",
	types.BlueprintTypeServices:      "Processing services",
	types.BlueprintTypeUsers:         "Processing users",
	types.BlueprintTypeGit:           "Processing git repositories",
	types.BlueprintTypeScripts:       "Processing scripts",
	types.BlueprintTypeSSHKeys:       "Processing ssh keys",
	types.BlueprintTypeFonts:         "Processing fonts",
	types.BlueprintTypeConfiguration: "Processing configurations",
}

// LogReporter reproduces the pre-event streaming output: headless runs
// (`rwr all > log`, CI=true, TERM=dumb) are byte-identical to before the
// event bus existed. It is the default reporter.
type LogReporter struct{}

// Emit renders an event exactly as the old inline code did.
func (LogReporter) Emit(event Event) {
	switch e := event.(type) {
	case ProcStarted:
		if label, ok := processorLabels[e.Processor]; ok {
			log.Info(label)
		} else {
			log.Infof("Processing %s", e.Processor)
		}
	case ProcSkipped:
		log.Warnf("Unknown processor: %s", e.Processor)
	case TerminalReq:
		// The pre-event direct wiring: the command owns the real terminal.
		// stderr is deliberately not captured (sudo's prompt lives there).
		if e.Cmd.Stdin == nil {
			e.Cmd.Stdin = os.Stdin
		}
		e.Cmd.Stdout = os.Stdout
		e.Cmd.Stderr = os.Stderr
		e.Done <- e.Cmd.Run()
	case TerminalFunc:
		// Headless the terminal is already free; just run the interaction.
		e.Done <- e.Run()
	case HaltReq:
		// Headless interactive keeps its historical behavior: the first
		// processor error aborts the run.
		e.Decision <- HaltAbort
	case ProcFinished, LaneUpdate, ResourceDone, RunFinished:
		// The streaming output never printed these as their own lines; the
		// processors' own log calls carry the detail.
	}
}

// current is the active reporter. Package state mirrors the executor seam in
// internal/system: the run loop and runCommand emit without threading a
// reporter through every processor signature.
var current Reporter = LogReporter{}

// Set installs a reporter and returns a restore func (test/TUI seam).
func Set(r Reporter) (restore func()) {
	previous := current
	current = r
	return func() { current = previous }
}

// Emit sends an event to the active reporter.
func Emit(event Event) { current.Emit(event) }

// RequestHalt reports an interactive processor error and blocks until the
// operator (or the LogReporter's abort default) decides what happens next.
func RequestHalt(processor string, err error) HaltDecision {
	decision := make(chan HaltDecision, 1)
	Emit(HaltReq{Processor: processor, Err: err, Decision: decision})
	return <-decision
}

// WithTerminal runs fn with exclusive use of the real terminal. Headless it
// runs fn directly; under the TUI the dashboard suspends itself first and
// resumes after. Every in-process prompt (huh forms, raw stdin reads) MUST go
// through this: a prompt that reads stdin while the dashboard owns it
// deadlocks the run - the prompt never gets keystrokes, the dashboard keeps
// eating them, and ctrl-c dies with both.
func WithTerminal(fn func() error) error {
	done := make(chan error, 1)
	Emit(TerminalFunc{Run: fn, Done: done})
	return <-done
}
