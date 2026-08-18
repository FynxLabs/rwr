package tui

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"golang.org/x/term"
)

// Active reports whether the TUI should run. Anything short of a real,
// capable, non-CI terminal falls back to the LogReporter, whose output is
// byte-identical to the pre-TUI stream - `rwr all > install.log` already
// fails the TTY check and behaves as today.
func Active(noTUI bool) bool {
	if noTUI {
		return false
	}
	if os.Getenv("CI") != "" {
		// Some runners allocate a pty and would otherwise get escape
		// sequences in a build log.
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// recognizedTerminal gates the OSC 9 notification: unknown terminals
// sometimes print unrecognized OSC payloads as literal text.
func recognizedTerminal() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	termProgram := os.Getenv("TERM_PROGRAM")
	for _, known := range []string{"iTerm.app", "WezTerm", "ghostty", "Apple_Terminal", "kitty", "alacritty"} {
		if strings.EqualFold(termProgram, known) {
			return true
		}
	}
	for _, env := range []string{"KITTY_WINDOW_ID", "ALACRITTY_WINDOW_ID", "GHOSTTY_RESOURCES_DIR", "WEZTERM_PANE"} {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

// Reporter adapts the event bus onto a running program.
type Reporter struct {
	program *tea.Program
}

// NewReporter wraps p.Send.
func NewReporter(program *tea.Program) *Reporter { return &Reporter{program: program} }

// Emit implements reporting.Reporter.
func (r *Reporter) Emit(e reporting.Event) { r.program.Send(event{e: e}) }

// SupportsInlinePrompts lets processors ask through the running model instead
// of suspending Bubble Tea and competing for stdin.
func (r *Reporter) SupportsInlinePrompts() bool { return true }
