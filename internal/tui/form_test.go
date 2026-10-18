package tui

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/freehold-digital/rwr/internal/prompts"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

type inspectForm func(*Model)

// Use the real event loop: huh emits follow-up commands when submitting and
// validating, so merely calling key() would miss breakouts and lost commands.
type formTestModel struct {
	*Model
	t *testing.T
}

func (m *formTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if inspect, ok := msg.(inspectForm); ok {
		inspect(m.Model)
		return m, nil
	}
	_, cmd := m.Model.Update(msg)
	if m.state == Suspended {
		m.t.Error("form handed the terminal away from the dashboard")
	}
	return m, cmd
}

func startFormProgram(t *testing.T) *tea.Program {
	t.Helper()
	m := New(mustTheme("rwr"), &types.Plan{Order: []string{"ssh_keys"}}, reporting.NewStore(10), false, "")
	p := tea.NewProgram(&formTestModel{Model: m, t: t}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(100, 30))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	restore := reporting.Set(NewReporter(p))
	reporting.SetCommandSink(m.store)
	lost := make(chan struct{})
	restoreLost := reporting.SetTerminalLost(lost)
	t.Cleanup(func() {
		close(lost)
		p.Quit()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("dashboard failed to exit")
		}
		restoreLost()
		restore()
		reporting.SetCommandSink(nil)
	})
	return p
}

func waitFormView(t *testing.T, p *tea.Program, text string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		seen := make(chan bool, 1)
		p.Send(inspectForm(func(m *Model) {
			view := m.View()
			seen <- m.state == Prompting && view.AltScreen && strings.Contains(view.Content, text)
		}))
		select {
		case ok := <-seen:
			if ok {
				return
			}
		case <-deadline:
			t.Fatalf("popup never displayed %q", text)
		}
		select {
		case <-deadline:
			t.Fatalf("popup never displayed %q", text)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestGitHubAuthOptionsStayInDashboard(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan prompts.GitHubAuthChoice, 1)
	go func() {
		choice, err := prompts.PromptGitHubAuthMethod()
		if err != nil {
			t.Error(err)
		}
		result <- choice
	}()
	for _, label := range []string{"GitHub authentication required", "OAuth (recommended)", "Enter GitHub token manually", "Skip (don't upload"} {
		waitFormView(t, p, label)
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case choice := <-result:
		if choice != prompts.GitHubAuthSkip {
			t.Fatalf("choice = %q", choice)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("choice was not delivered; embedded submission stalled or exited the dashboard")
	}
}

func TestSudoPromptDuringGitHubFormOwnsVisibleInput(t *testing.T) {
	p := startFormProgram(t)
	choice := make(chan prompts.GitHubAuthChoice, 1)
	go func() {
		got, err := prompts.PromptGitHubAuthMethod()
		if err != nil {
			t.Error(err)
		}
		choice <- got
	}()
	waitFormView(t, p, "GitHub authentication required")
	password := make(chan reporting.SecretResult, 1)
	go func() {
		value, err := reporting.RequestSecret("sudo password")
		password <- reporting.SecretResult{Value: value, Err: err}
	}()
	waitFormView(t, p, "sudo password")
	for _, r := range "fixture-" {
		p.Send(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	p.Send(tea.PasteMsg{Content: "password"})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case got := <-password:
		if got.Err != nil || string(got.Value) != "fixture-password" {
			t.Fatal("visible sudo popup did not receive its password")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("visible sudo popup lost input to the GitHub form")
	}
	waitFormView(t, p, "GitHub authentication required")
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyDown})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case got := <-choice:
		if got != prompts.GitHubAuthSkip {
			t.Fatal("sudo password input changed the GitHub choice")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("GitHub form did not resume after the sudo popup")
	}
}

func TestOAuthCompletionDoesNotHidePendingSudoPrompt(t *testing.T) {
	p := startFormProgram(t)
	authorized := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- reporting.RunFormWithTask(huh.NewForm(huh.NewGroup(huh.NewNote().Title("Waiting for GitHub").Next(true))), func(ctx context.Context) error {
			select {
			case <-authorized:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	waitFormView(t, p, "Waiting for GitHub")
	password := make(chan reporting.SecretResult, 1)
	go func() {
		value, err := reporting.RequestSecret("sudo password")
		password <- reporting.SecretResult{Value: value, Err: err}
	}()
	waitFormView(t, p, "sudo password")
	close(authorized)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OAuth task did not complete")
	}
	waitFormView(t, p, "sudo password")
	p.Send(tea.KeyPressMsg{Code: tea.KeyEscape})
	select {
	case got := <-password:
		if !errors.Is(got.Err, reporting.ErrPromptCancelled) {
			t.Fatalf("sudo cancellation = %v", got.Err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sudo prompt became unresponsive after OAuth completion")
	}
}

func TestPackageManagerChoiceCancellationReturnsNoDefault(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, {Code: 'c', Mod: tea.ModCtrl}} {
		t.Run(key.String(), func(t *testing.T) {
			p := startFormProgram(t)
			type answer struct {
				choice string
				err    error
			}
			result := make(chan answer, 1)
			go func() {
				choice, err := system.PromptUserChoice("Choose a package manager", []string{"brew", "nix"}, "brew")
				result <- answer{choice, err}
			}()
			waitFormView(t, p, "Choose a package manager")
			p.Send(key)
			select {
			case got := <-result:
				if got.choice != "" || !errors.Is(got.err, reporting.ErrPromptCancelled) {
					t.Fatalf("cancel returned choice=%q error=%v", got.choice, got.err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("package manager choice did not cancel")
			}
		})
	}
}

func TestGitHubTokenPopupMasksAndValidatesPaste(t *testing.T) {
	p := startFormProgram(t)
	result := make(chan string, 1)
	go func() {
		token, err := prompts.PromptGitHubToken()
		if err != nil {
			t.Error(err)
		}
		result <- token
	}()
	waitFormView(t, p, "personal access token")
	p.Send(tea.PasteMsg{Content: "ghp_x"})
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFormView(t, p, "not a whole token")
	for range "ghp_x" {
		p.Send(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	token := "github_pat_" + strings.Repeat("a", 25)
	p.Send(tea.PasteMsg{Content: token})
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) { checked <- strings.Contains(m.View().Content, token) }))
	if <-checked {
		t.Fatal("pasted GitHub token appeared in the popup")
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	select {
	case got := <-result:
		if got != token {
			t.Fatal("popup did not return the pasted token")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("token submission stalled")
	}
}

func TestOAuthPopupWaitsForTaskAndCancels(t *testing.T) {
	p := startFormProgram(t)
	started, cancelled := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		form := huh.NewForm(huh.NewGroup(huh.NewNote().Title("GitHub authentication").Description("Visit github.com/login/device\nEnter code: ABCD-1234").Next(true)))
		result <- reporting.RunFormWithTask(form, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		})
	}()
	waitFormView(t, p, "ABCD-1234")
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("OAuth task did not start")
	}
	p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	waitFormView(t, p, "ABCD-1234")
	p.Send(tea.KeyPressMsg{Code: tea.KeyEscape})
	select {
	case err := <-result:
		if !errors.Is(err, reporting.ErrPromptCancelled) {
			t.Fatalf("cancel = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OAuth dialog did not cancel")
	}
	select {
	case <-cancelled:
	case <-time.After(3 * time.Second):
		t.Fatal("OAuth polling outlived its dialog")
	}
}

func TestOAuthPopupCompletesWithoutQuittingDashboard(t *testing.T) {
	p := startFormProgram(t)
	authorized := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		form := huh.NewForm(huh.NewGroup(huh.NewNote().Title("Waiting for GitHub authorization").Next(true)))
		result <- reporting.RunFormWithTask(form, func(ctx context.Context) error {
			select {
			case <-authorized:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	waitFormView(t, p, "Waiting for GitHub authorization")
	close(authorized)
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("authorization did not complete")
	}
	checked := make(chan bool, 1)
	p.Send(inspectForm(func(m *Model) { checked <- m.state == Running && m.form == nil }))
	if !<-checked {
		t.Fatal("OAuth completion did not resume the dashboard")
	}
}
