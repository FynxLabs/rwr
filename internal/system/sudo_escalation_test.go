package system

import (
	"runtime"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
)

// The guard decides which commands get the terminal when sudo is not already
// cached, so that a password prompt the work itself makes is visible rather
// than hidden behind the dashboard. It never causes rwr to ask on its own
// account.
func TestSudoTerminalHandoverCoversCommandsThatEscalateThemselves(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	tests := []struct {
		name string
		cmd  types.Command
		want bool
	}{
		{
			name: "rwr elevates it",
			cmd:  types.Command{Exec: "pacman", Elevated: true},
			want: true,
		},
		{
			name: "rwr runs it as another user",
			cmd:  types.Command{Exec: "makepkg", AsUser: "builder"},
			want: true,
		},
		{
			// The case that hung a real run: brew must not be elevated, and
			// still shells out to sudo for a cask install.
			name: "unprivileged but escalates itself",
			cmd:  types.Command{Exec: "brew", Escalates: true},
			want: true,
		},
		{
			name: "plain unprivileged command",
			cmd:  types.Command{Exec: "go"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mayPromptForSudo(tt.cmd); got != tt.want {
				t.Errorf("mayPromptForSudo = %v, want %v", got, tt.want)
			}
		})
	}
}

// rwr must never ask for a password on its own account.
//
// It used to run `sudo -v` before any command that might escalate, which meant
// a prompt per minute through a run of ordinary formulae that never touch
// root. The probe replacing it uses -n, which answers from the cache or fails
// and never prompts.
func TestSudoProbeNeverPrompts(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	// Whatever the machine's cache state, the probe has to return promptly
	// rather than sit on a password prompt.
	done := make(chan bool, 1)
	go func() { done <- sudoCredentialsCached() }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the sudo probe blocked; it must never prompt")
	}
}
