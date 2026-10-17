package reporting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/huh/v2"
)

type formCapture struct{ requests chan FormReq }

func (r formCapture) SupportsInlinePrompts() bool { return true }
func (r formCapture) Emit(e Event) {
	if req, ok := e.(FormReq); ok {
		r.requests <- req
	}
}

func TestFormTerminalLossCancelsWithoutTerminalFallback(t *testing.T) {
	requests := make(chan FormReq, 1)
	defer Set(formCapture{requests})()
	lost := make(chan struct{})
	defer SetTerminalLost(lost)()
	result := make(chan error, 1)
	go func() {
		result <- RunForm(huh.NewForm(huh.NewGroup(huh.NewInput().Title("Token"))))
	}()
	var req FormReq
	select {
	case req = <-requests:
	case <-time.After(time.Second):
		t.Fatal("no inline request")
	}
	close(lost)
	select {
	case err := <-result:
		if !errors.Is(err, ErrPromptUnavailable) {
			t.Fatalf("terminal loss = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("form hung after terminal loss")
	}
	if !req.Claim.Load() {
		t.Fatal("abandoned form can still be answered")
	}
	if !errors.Is(req.Context.Err(), context.Canceled) {
		t.Fatal("form context was not canceled")
	}
}

func TestWaitingFormHasInstructionsWithoutRunningAProgram(t *testing.T) {
	form := huh.NewForm(huh.NewGroup(huh.NewNote().Title("GitHub authentication").Description("Visit github.com/login/device\nCode: ABCD-1234").Next(true)))
	form.Init()
	if !strings.Contains(form.View(), "ABCD-1234") {
		t.Fatal("headless OAuth instructions would be empty")
	}
}

func TestRawTerminalCallbackRefusedInsideDashboard(t *testing.T) {
	requests := make(chan FormReq, 1)
	defer Set(formCapture{requests})()
	called := false
	err := WithTerminal(func() error { called = true; return nil })
	if called || !errors.Is(err, ErrPromptUnavailable) {
		t.Fatalf("raw terminal callback was allowed: called=%v err=%v", called, err)
	}
}
