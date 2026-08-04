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
