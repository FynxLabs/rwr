//go:build !windows

package tui

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func TestExternalToolUsesPopupPTYWithPrivateMaskedOutput(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan error, 1)
	// This fixture requires a real controlling terminal, disables echo for the
	// answer, then deliberately emits a known secret to exercise redaction.
	cmd := exec.Command("sh", "-c", `test -t 0 && test -t 1 || exit 2
stty -echo
printf 'Enter verification code: '
IFS= read -r answer
test "$answer" = 'popup-secret' || exit 3
printf '\nReceived popup-secret\nPress Enter to finish: '
IFS= read -r finish
stty echo`)
	go func() {
		result <- reporting.RunInteractiveCommand(context.Background(), cmd, []string{"popup-secret"}, true)
	}()
	waitFormView(t, p, "Enter verification code")
	p.Send(tea.WindowSizeMsg{Width: 80, Height: 28})
	p.Send(tea.PasteMsg{Content: "popup-secret"})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFormView(t, p, "Received [redacted]")
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) {
		view := m.View().Content
		logs := m.store.Records(m.store.NewView(""))
		checked <- !strings.Contains(view, "popup-secret") && len(logs) == 0
	}))
	if !<-checked {
		t.Fatal("private tool output leaked to the popup or log store")
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tool did not complete inside its popup")
	}
}

func TestExternalToolPopupCancellationStopsChild(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan error, 1)
	cmd := exec.Command("sh", "-c", "printf 'Waiting for external authorization'; sleep 30 & wait")
	go func() { result <- reporting.RunInteractiveCommand(context.Background(), cmd, nil, true) }()
	waitFormView(t, p, "Waiting for external authorization")
	p.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	select {
	case err := <-result:
		if !errors.Is(err, reporting.ErrPromptCancelled) {
			t.Fatalf("cancel = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("canceled tool and its child kept running")
	}
}

func TestProductionExecutorPreservesPrivateSuppliedInput(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan error, 1)
	go func() {
		result <- system.RunCommand(types.Command{Exec: "sh", Interactive: true, Stdin: "prefed-private-value\n", Args: []string{"-c", `
IFS= read -r supplied
test "$supplied" = 'prefed-private-value' || exit 2
printf 'Supplied input accepted; type continue: '
IFS= read -r answer
test "$answer" = 'continue' || exit 3
`}}, false)
	}()
	waitFormView(t, p, "Supplied input accepted")
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) { checked <- strings.Contains(m.View().Content, "prefed-private-value") }))
	if <-checked {
		t.Fatal("pre-fed input was echoed by the PTY")
	}
	p.Send(tea.PasteMsg{Content: "continue"})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("production executor did not preserve popup input after supplied stdin")
	}
}
