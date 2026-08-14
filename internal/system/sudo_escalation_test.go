package system

import (
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// The guard decides whether sudo's credential cache is warmed before a command
// runs. Getting it wrong is not a cosmetic failure: an un-warmed command that
// calls sudo internally prompts on /dev/tty, behind the dashboard, and the run
// hangs with no visible prompt.
func TestSudoValidationCoversCommandsThatEscalateThemselves(t *testing.T) {
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
			if got := wantsSudoCredentials(tt.cmd); got != tt.want {
				t.Errorf("wantsSudoCredentials = %v, want %v", got, tt.want)
			}
		})
	}
}
