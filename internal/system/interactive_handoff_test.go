package system

import (
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

type recordingReporter struct {
	events []reporting.Event
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
