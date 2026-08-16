// Package reporting is the event bus between the run loop and whatever is
// watching it. All() and the processors do not know a TUI exists: they emit
// events; a TUIReporter forwards them to the program, and LogReporter
// reproduces the pre-event streaming output exactly (the byte-identical
// headless contract).
package reporting

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// Reporter consumes run events.
type Reporter interface {
	Emit(Event)
}

// SupportsInlinePrompts reports whether the active display can collect input
// without releasing the terminal. The Bubble Tea reporter supports this;
// LogReporter deliberately leaves the established terminal fallback intact.
func SupportsInlinePrompts() bool {
	currentMu.RLock()
	r := current
	currentMu.RUnlock()
	provider, ok := r.(interface{ SupportsInlinePrompts() bool })
	return ok && provider.SupportsInlinePrompts()
}

var ErrPromptUnavailable = errors.New("inline prompt unavailable")
var ErrPromptCancelled = errors.New("prompt cancelled")

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
	// Claim makes the request run-once: the servicer and the waiter's
	// terminal-lost fallback CAS it, and only the winner executes. Without
	// it, a quit racing the exec callback could run the request twice.
	Claim *atomic.Bool
}

// TerminalFunc asks the display layer to lend the real terminal to an
// in-process interaction - a huh form, a raw stdin prompt. The TerminalReq
// counterpart for code that runs in this process rather than a child.
type TerminalFunc struct {
	Run   func() error
	Done  chan error
	Claim *atomic.Bool
}

type SecretResult struct {
	Value []byte
	Err   error
}

// SecretReq asks the active display to collect a masked value without handing
// away the terminal. It is used only when SupportsInlinePrompts is true.
type SecretReq struct {
	Prompt string
	Result chan SecretResult
	Claim  *atomic.Bool
}

type ConfirmResult struct {
	Yes bool
	Err error
}

// ConfirmReq asks a yes/no question inside the active display.
type ConfirmReq struct {
	Prompt string
	Result chan ConfirmResult
	Claim  *atomic.Bool
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
	Retryable bool
	Decision  chan HaltDecision
	// Claim is CAS'd by whoever answers - the operator's keypress or the
	// terminal-lost fallback - so a decision is made exactly once.
	Claim *atomic.Bool
}

// TryClaim reports whether the caller won the right to service a claimable
// request. A nil claim (an event constructed directly, e.g. in tests) is
// always claimable.
func TryClaim(claim *atomic.Bool) bool {
	return claim == nil || claim.CompareAndSwap(false, true)
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
func (SecretReq) runEvent()    {}
func (ConfirmReq) runEvent()   {}
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
		if !TryClaim(e.Claim) {
			return
		}
		if e.Cmd.Stdin == nil {
			e.Cmd.Stdin = os.Stdin
		}
		e.Cmd.Stdout = os.Stdout
		e.Cmd.Stderr = os.Stderr
		e.Done <- e.Cmd.Run()
	case TerminalFunc:
		// Headless the terminal is already free; just run the interaction.
		if !TryClaim(e.Claim) {
			return
		}
		e.Done <- e.Run()
	case SecretReq:
		if TryClaim(e.Claim) {
			e.Result <- SecretResult{Err: ErrPromptUnavailable}
		}
	case ConfirmReq:
		if TryClaim(e.Claim) {
			e.Result <- ConfirmResult{Err: ErrPromptUnavailable}
		}
	case HaltReq:
		// Headless interactive keeps its historical behavior: the first
		// processor error aborts the run.
		if !TryClaim(e.Claim) {
			return
		}
		e.Decision <- HaltAbort
	case ProcFinished, LaneUpdate, ResourceDone, RunFinished:
		// The streaming output never printed these as their own lines; the
		// processors' own log calls carry the detail.
	}
}

func RequestSecret(prompt string) ([]byte, error) {
	if !SupportsInlinePrompts() {
		return nil, ErrPromptUnavailable
	}
	result := make(chan SecretResult, 1)
	claim := &atomic.Bool{}
	Emit(SecretReq{Prompt: prompt, Result: result, Claim: claim})
	select {
	case answer := <-result:
		return answer.Value, answer.Err
	case <-TerminalLost():
		if claim.CompareAndSwap(false, true) {
			return nil, ErrPromptUnavailable
		}
		answer := <-result
		return answer.Value, answer.Err
	}
}

func RequestConfirmation(prompt string) (bool, error) {
	if !SupportsInlinePrompts() {
		return false, ErrPromptUnavailable
	}
	result := make(chan ConfirmResult, 1)
	claim := &atomic.Bool{}
	Emit(ConfirmReq{Prompt: prompt, Result: result, Claim: claim})
	select {
	case answer := <-result:
		return answer.Yes, answer.Err
	case <-TerminalLost():
		if claim.CompareAndSwap(false, true) {
			return false, ErrPromptUnavailable
		}
		answer := <-result
		return answer.Yes, answer.Err
	}
}

// current is the active reporter. Package state mirrors the executor seam in
// internal/system: the run loop and runCommand emit without threading a
// reporter through every processor signature. Guarded by a mutex because the
// TUI runner swaps it back to the LogReporter mid-run when the dashboard
// exits early - the run goroutine reads it on every Emit, and an interface
// value is two words, so an unsynchronized swap can tear.
var (
	currentMu sync.RWMutex
	current   Reporter = LogReporter{}
)

// Set installs a reporter and returns a restore func (test/TUI seam).
func Set(r Reporter) (restore func()) {
	currentMu.Lock()
	previous := current
	current = r
	currentMu.Unlock()
	return func() {
		currentMu.Lock()
		current = previous
		currentMu.Unlock()
	}
}

// Emit sends an event to the active reporter.
func Emit(event Event) {
	currentMu.RLock()
	r := current
	currentMu.RUnlock()
	r.Emit(event)
}

// terminalLost, when set, is closed by the TUI runner the moment the
// dashboard program exits. Bubbletea silently drops Sends once its context
// is cancelled, so a blocking event emitted in the window between program
// exit and the reporter swap would leave its waiter blocked forever - the
// waiters below select on this and fall back to the headless behavior.
var (
	terminalLostMu sync.RWMutex
	terminalLost   chan struct{}
)

// SetTerminalLost installs the channel the TUI runner closes on program
// exit (nil clears). Returns a restore func.
func SetTerminalLost(ch chan struct{}) (restore func()) {
	terminalLostMu.Lock()
	previous := terminalLost
	terminalLost = ch
	terminalLostMu.Unlock()
	return func() {
		terminalLostMu.Lock()
		terminalLost = previous
		terminalLostMu.Unlock()
	}
}

// TerminalLost returns the current lost-channel; nil (which never fires in
// a select) when no dashboard is running.
func TerminalLost() <-chan struct{} {
	terminalLostMu.RLock()
	defer terminalLostMu.RUnlock()
	return terminalLost
}

// RequestHalt reports an interactive processor error and blocks until the
// operator (or the LogReporter's abort default) decides what happens next.
// If the dashboard dies before answering, the headless default (abort) is
// taken rather than blocking forever on a dropped Send.
func RequestHalt(processor string, err error) HaltDecision {
	return requestHalt(processor, err, true)
}

// RequestFinalHalt reports failures collected after all processors have run.
// Retrying is deliberately unavailable because replaying the entire run could
// repeat operations that already succeeded.
func RequestFinalHalt(err error) HaltDecision {
	return requestHalt("run", err, false)
}

func requestHalt(processor string, err error, retryable bool) HaltDecision {
	decision := make(chan HaltDecision, 1)
	claim := &atomic.Bool{}
	Emit(HaltReq{Processor: processor, Err: err, Retryable: retryable, Decision: decision, Claim: claim})
	select {
	case d := <-decision:
		return d
	case <-TerminalLost():
		// Whoever wins the claim answers; losing it means an answer is
		// already on its way (the claimant writes the buffered channel
		// immediately after claiming), so waiting is safe.
		if claim.CompareAndSwap(false, true) {
			return HaltAbort
		}
		return <-decision
	}
}

// WithTerminal runs fn with exclusive use of the real terminal. Headless it
// runs fn directly; under the TUI the dashboard suspends itself first and
// resumes after. Every in-process prompt (huh forms, raw stdin reads) MUST go
// through this: a prompt that reads stdin while the dashboard owns it
// deadlocks the run - the prompt never gets keystrokes, the dashboard keeps
// eating them, and ctrl-c dies with both.
func WithTerminal(fn func() error) error {
	done := make(chan error, 1)
	claim := &atomic.Bool{}
	Emit(TerminalFunc{Run: fn, Done: done, Claim: claim})
	select {
	case err := <-done:
		return err
	case <-TerminalLost():
		// Run-once via the claim: if the dashboard already claimed this
		// request, its exec callback runs independently of the program's
		// lifetime and will write done - wait for it. Winning the claim
		// means nobody ran fn; the terminal is free now, run it directly.
		if claim.CompareAndSwap(false, true) {
			return fn()
		}
		return <-done
	}
}
