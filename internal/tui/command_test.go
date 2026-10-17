//go:build !windows

package tui

import (
	"context"
	"errors"
	"io"
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
		result <- system.RunCommand(types.Command{Exec: "sh", Interactive: true, Stdin: "prefed-private-value\nprefed-second-value\n", Args: []string{"-c", `
test -t 0 || exit 4
IFS= read -r supplied
test "$supplied" = 'prefed-private-value' || exit 2
IFS= read -r second
test "$second" = 'prefed-second-value' || exit 5
stty -echo
printf 'Supplied input accepted; enter password: '
IFS= read -r answer
test "$answer" = 'later-unknown-password' || exit 3
stty echo
printf '\nPassword accepted\nPress Enter to finish: '
IFS= read -r finish
`}}, false)
	}()
	waitFormView(t, p, "Supplied input accepted")
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) { checked <- strings.Contains(m.View().Content, "prefed-") }))
	if <-checked {
		t.Fatal("pre-fed input was echoed by the PTY")
	}
	p.Send(tea.PasteMsg{Content: "later-unknown-password"})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFormView(t, p, "Password accepted")
	p.Send(inspectForm(func(m *Model) { checked <- strings.Contains(m.View().Content, "later-unknown-password") }))
	if <-checked {
		t.Fatal("later password was echoed despite the child's stdin echo setting")
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("production executor did not preserve popup input after supplied stdin")
	}
	p.Send(inspectForm(func(m *Model) {
		captured := false
		for _, record := range m.store.Records(m.store.NewView("")) {
			if strings.Contains(record.Msg, "Password accepted") {
				captured = true
			}
			if strings.Contains(record.Msg, "prefed-") || strings.Contains(record.Msg, "later-unknown-password") {
				checked <- false
				return
			}
		}
		checked <- captured
	}))
	if !<-checked {
		t.Fatal("command log capture was missing or persisted a password")
	}
}

func TestInvalidInitialInputRefusedBeforeChildStarts(t *testing.T) {
	t.Parallel()
	for name, input := range map[string]string{
		"unterminated":     "private-input",
		"oversized":        strings.Repeat("x", 4096) + "\n",
		"terminal control": "private\x03input\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("sh", "-c", "exit 0")
			cmd.Stdin = strings.NewReader(input)
			child, err := startPopupPTY(context.Background(), cmd, 56, 16)
			if child != nil {
				closePopupIO(child)
			}
			if err == nil || cmd.Process != nil {
				t.Fatalf("invalid initial input launched a child: process=%v error=%v", cmd.Process, err)
			}
			if strings.Contains(err.Error(), "private") {
				t.Fatal("invalid input appeared in the error")
			}
		})
	}
}

type popupInputReader func([]byte) (int, error)

func (r popupInputReader) Read(p []byte) (int, error) { return r(p) }

func TestCanceledInitialInputDoesNotStartChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.Command("sh", "-c", "exit 0")
	cmd.Stdin = popupInputReader(func(p []byte) (int, error) { cancel(); return copy(p, "initial\n"), io.EOF })
	child, err := startPopupPTY(ctx, cmd, 56, 16)
	if child != nil {
		closePopupIO(child)
	}
	if !errors.Is(err, context.Canceled) || cmd.Process != nil {
		t.Fatalf("canceled staging launched a child: process=%v error=%v", cmd.Process, err)
	}
}

func TestToolThatDoesNotReadCannotFreezeDashboardOnPaste(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan error, 1)
	cmd := exec.Command("sh", "-c", "stty -echo; printf 'Ready for oversized input'; sleep 30")
	go func() { result <- reporting.RunInteractiveCommand(context.Background(), cmd, nil, true) }()
	waitFormView(t, p, "Ready for oversized input")
	p.Send(tea.PasteMsg{Content: strings.Repeat("x", 1<<20)})
	select {
	case err := <-result:
		if !errors.Is(err, reporting.ErrPromptCancelled) {
			t.Fatalf("overflow = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("unread tool input froze the dashboard")
	}
}
