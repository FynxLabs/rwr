//go:build !windows

package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/freehold-digital/rwr/internal/prompts"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func TestWarmSudoProbeCannotSuppressPasswordPrompt(t *testing.T) {
	for _, mode := range []string{"captured", "interactive", "output"} {
		for _, policy := range []string{"expired", "cached", "cancel"} {
			t.Run(mode+"/"+policy, func(t *testing.T) {
				dir := t.TempDir()
				// The ambient probe is warm, but the command's session/TTY has no
				// reusable ticket. Noninteractive sudo refuses without prompting.
				if err := os.WriteFile(filepath.Join(dir, "sudo"), []byte(`#!/bin/sh
case "$1" in
  -n) printf 'sudo: a password is required\n' >&2; exit 1 ;;
  -S)
    IFS= read -r password
    [ "$password" = 'warm-probe-password' ] || exit 2
    [ "$4" = '-v' ] && exit 0
    exit 3 ;;
  -A)
    if [ "$RWR_TEST_SUDO_POLICY" != 'cached' ]; then
      password=$("$SUDO_ASKPASS") || exit 4
      [ "$password" = 'warm-probe-password' ] || exit 5
    fi
    shift
    [ "$1" = '--' ] || exit 6
    shift
    exec "$@" ;;
  *) exit 7 ;;
esac
`), 0700); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				release := system.BeginRun()
				t.Cleanup(release)
				restoreProbe := system.SetSudoProbeForTest(func(context.Context) error { return nil })
				t.Cleanup(restoreProbe)
				p := startFormProgram(t)
				result := make(chan error, 1)
				go func() {
					choice, err := prompts.PromptGitHubAuthMethod()
					if err == nil && choice != prompts.GitHubAuthSkip {
						err = errors.New("unexpected GitHub choice")
					}
					if err == nil {
						cmd := types.Command{Exec: "sh", Args: []string{"-c", "IFS= read -r supplied; test \"$supplied\" = child-input || exit 8; printf 'privileged-command-ran\\n'"}, Stdin: "child-input\n", Elevated: true, Interactive: mode == "interactive", Variables: map[string]string{"RWR_TEST_SUDO_POLICY": policy}}
						if mode == "output" {
							var output string
							output, err = system.RunCommandOutput(cmd, false)
							if err == nil && !strings.Contains(output, "privileged-command-ran") {
								err = errors.New("privileged command output missing")
							}
						} else {
							err = system.RunCommand(cmd, false)
						}
					}
					result <- err
				}()
				waitFormView(t, p, "GitHub authentication required")
				p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
				p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
				p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
				if policy != "cached" {
					waitFormView(t, p, "sudo password")
					if policy == "cancel" {
						p.Send(tea.KeyPressMsg{Code: tea.KeyEscape})
					} else {
						p.Send(tea.PasteMsg{Content: "warm-probe-password"})
						p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
					}
				}
				select {
				case err := <-result:
					if policy == "cancel" {
						if err == nil {
							t.Fatal("canceled sudo prompt ran the command")
						}
					} else if err != nil {
						t.Fatal(err)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("sudo failed to resume its actual command after authentication")
				}
			})
		}
	}
}

func TestProductionSudoPasswordAfterGitHubPopup(t *testing.T) {
	// Use the real executor and validation subprocess, with a local sudo
	// fixture so the test never changes host credentials or runs as root.
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "sudo"), []byte(`#!/bin/sh
[ "$1" = '-S' ] || exit 1
[ "$2" = '-p' ] && [ "$3" = '' ] || exit 2
IFS= read -r password
[ "$password" = 'sudo-fixture-password' ] || exit 3
shift 3
[ "$1" = '-v' ] && exit 0
[ "$1" = '--' ] || exit 4
shift
exec "$@"
`), 0700)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	release := system.BeginRun()
	t.Cleanup(release)
	p := startFormProgram(t)
	result := make(chan error, 1)
	go func() {
		choice, err := prompts.PromptGitHubAuthMethod()
		if err == nil && choice != prompts.GitHubAuthSkip {
			err = errors.New("unexpected GitHub choice")
		}
		if err == nil {
			err = system.RunCommand(types.Command{Exec: "sh", Args: []string{"-c", "printf 'sudo-command-completed\\n'"}, Elevated: true}, false)
		}
		result <- err
	}()
	waitFormView(t, p, "GitHub authentication required")
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFormView(t, p, "sudo password")
	p.Send(tea.PasteMsg{Content: "sudo-fixture-password"})
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) { checked <- strings.Contains(m.View().Content, "sudo-fixture-password") }))
	if <-checked {
		t.Fatal("sudo password was displayed")
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sudo validation or elevated command stalled after GitHub popup")
	}
	p.Send(inspectForm(func(m *Model) {
		completed := false
		for _, record := range m.store.Records(m.store.NewView("")) {
			if strings.Contains(record.Msg, "sudo-fixture-password") {
				checked <- false
				return
			}
			completed = completed || (record.Src == reporting.SrcStdout && strings.Contains(record.Msg, "sudo-command-completed"))
		}
		checked <- completed
	}))
	if !<-checked {
		t.Fatal("elevated command output was missing or contained the sudo password")
	}
}

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
