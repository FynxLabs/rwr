package reporting

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"charm.land/huh/v2"
)

// FormReq embeds a form in the active dashboard. Task, when present, keeps
// the dialog visible until background work finishes, with Esc to cancel.
type FormReq struct {
	Form    *huh.Form
	Task    func(context.Context) error
	Context context.Context
	Done    chan error
	Claim   *atomic.Bool
}

func (FormReq) runEvent() {}

func RunForm(form *huh.Form) error {
	return RunFormWithTask(form, nil)
}

func RunFormWithTask(form *huh.Form, task func(context.Context) error) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !SupportsInlinePrompts() {
		return WithTerminal(func() error {
			if task == nil {
				return form.Run()
			}
			form.Init() // builds the static waiting instructions; no stdin reader
			if _, err := fmt.Fprintln(os.Stdout, form.View()); err != nil {
				return err
			}
			return task(ctx)
		})
	}
	done := make(chan error, 1)
	claim := &atomic.Bool{}
	Emit(FormReq{Form: form, Task: task, Context: ctx, Done: done, Claim: claim})
	select {
	case err := <-done:
		return err
	case <-TerminalLost():
		if claim.CompareAndSwap(false, true) {
			return ErrPromptUnavailable
		}
		return <-done
	}
}
