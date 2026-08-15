package system

import (
	"errors"
	"runtime"
	"strings"
	"testing"

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
// -n is the whole contract: it makes sudo answer from the credential cache or
// fail, and never prompt. Asserting only that the probe returns quickly does
// not pin that - on a host that cannot prompt at all, a bare `sudo -v` also
// returns immediately, so such a test passes with -n removed.
func TestSudoProbeIsNonInteractive(t *testing.T) {
	t.Parallel()

	if got := strings.Join(sudoProbeArgs, " "); got != "sudo -n -v" {
		t.Fatalf("probe = %q, want exactly \"sudo -n -v\"", got)
	}
}

// A cold cache is reported as cold, and does not prompt on the way there.
func TestSudoCredentialsCachedWhenTheProbeFails(t *testing.T) {
	defer SetSudoProbeForTest(func() error { return errors.New("a password is required") })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if sudoCredentialsCached() {
		t.Error("a failing probe was read as a warm cache")
	}
}

// A warm cache is reported as warm, so the command runs captured and the
// dashboard is not torn down for nothing.
func TestSudoCredentialsCachedWhenTheProbeSucceeds(t *testing.T) {
	defer SetSudoProbeForTest(func() error { return nil })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if !sudoCredentialsCached() {
		t.Error("a succeeding probe was read as a cold cache")
	}
}

// The probe is throttled, so a run of a hundred packages does not spawn a
// hundred sudo processes to ask a question whose answer does not change second
// to second.
func TestSudoProbeIsThrottled(t *testing.T) {
	var calls int
	defer SetSudoProbeForTest(func() error {
		calls++
		return nil
	})()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	for range 10 {
		sudoCredentialsCached()
	}
	if calls != 1 {
		t.Errorf("probe ran %d times for 10 commands, want 1", calls)
	}
}

// A cold cache is not throttled into looking warm: the answer has to be
// re-asked until it changes, or the first cold command would be the only one
// ever given the terminal.
func TestSudoProbeRetriesWhileCold(t *testing.T) {
	var calls int
	defer SetSudoProbeForTest(func() error {
		calls++
		return errors.New("a password is required")
	})()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	for range 3 {
		if sudoCredentialsCached() {
			t.Fatal("a failing probe was read as a warm cache")
		}
	}
	if calls != 3 {
		t.Errorf("probe ran %d times, want one per call while the cache is cold", calls)
	}
}
