package reporting

import (
	"context"
	"os/exec"
	"sync/atomic"
)

// RunInteractiveCommand gives a tool an embedded terminal popup, or the real
// terminal in headless mode. Private authentication output never enters logs.
func RunInteractiveCommand(ctx context.Context, command *exec.Cmd, secrets []string, private bool) error {
	return RunInteractiveCommandWithStart(ctx, command, secrets, private, nil)
}

// RunInteractiveCommandWithStart allows the executor to bind private askpass
// access to the actual child session before serving its authentication requests.
func RunInteractiveCommandWithStart(ctx context.Context, command *exec.Cmd, secrets []string, private bool, started func(*exec.Cmd)) error {
	inline := SupportsInlinePrompts()
	done := make(chan error, 1)
	claim := &atomic.Bool{}
	Emit(TerminalReq{Cmd: command, Context: ctx, Done: done, Claim: claim, Secrets: secrets, Private: private, Started: started})
	select {
	case err := <-done:
		return err
	case <-TerminalLost():
		if claim.CompareAndSwap(false, true) {
			if inline {
				return ErrPromptUnavailable
			}
			LogReporter{}.Emit(TerminalReq{Cmd: command, Done: done, Secrets: secrets, Private: private, Started: started})
		}
		return <-done
	}
}
