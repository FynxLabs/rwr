package system

import (
	"errors"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/types"
)

type recordingReporter struct {
	events []reporting.Event
}

type failingInteractiveReporter struct {
	diagnostic string
}

func (r failingInteractiveReporter) Emit(e reporting.Event) {
	req, ok := e.(reporting.TerminalReq)
	if !ok {
		return
	}
	if req.Diagnostic != nil {
		_, _ = io.WriteString(req.Diagnostic, r.diagnostic)
	}
	req.Done <- errors.New("exit status 1")
}

func (r *recordingReporter) Emit(e reporting.Event) {
	r.events = append(r.events, e)
	if req, ok := e.(reporting.TerminalReq); ok {
		// Behave like a display layer: run the child and report back.
		req.Done <- req.Cmd.Run()
	}
}

// Interactive commands are handed to the display layer via TerminalReq; the
// run blocks on Done. This is the seam a TUI suspends around - and it must
// fire regardless of the global interactive flag, because a single blueprint
// item can set interactive: true inside a non-interactive run.
func TestRunCommand_InteractiveGoesThroughTerminalReq(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/true")
	}
	rec := &recordingReporter{}
	defer reporting.Set(rec)()

	err := RunCommand(types.Command{Exec: "true", Interactive: true}, false)
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}

	found := false
	for _, e := range rec.events {
		if _, ok := e.(reporting.TerminalReq); ok {
			found = true
		}
	}
	if !found {
		t.Fatalf("no TerminalReq emitted; events: %#v", rec.events)
	}
}

func TestRunCommand_InteractiveFailureRetainsManagerDiagnostic(t *testing.T) {
	reporter := failingInteractiveReporter{
		diagnostic: "error: failed to synchronize all databases (unable to lock database)\n",
	}
	defer reporting.Set(reporter)()

	err := RunCommand(types.Command{Exec: "unused", Interactive: true}, false)
	if err == nil {
		t.Fatal("RunCommand returned nil for interactive failure")
	}
	if !strings.Contains(err.Error(), "unable to lock database") {
		t.Fatalf("RunCommand error = %v, want retained diagnostic", err)
	}
	if !IsPackageManagerUnavailable(err) {
		t.Fatalf("interactive lock failure was not classified as package-manager unavailable: %v", err)
	}
}
