package system

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/reporting"
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

func TestSudoValidationKeepsPasswordOutOfArgv(t *testing.T) {
	t.Parallel()

	password := []byte("correct horse battery staple\n")
	command := sudoValidationCommand(context.Background(), password)
	if want := []string{"sudo", "-S", "-p", "", "-v"}; !slices.Equal(command.Args, want) {
		t.Fatalf("validation argv = %q, want %q", command.Args, want)
	}
	for _, arg := range command.Args {
		if strings.Contains(arg, "correct horse") {
			t.Fatal("password appeared in sudo argv")
		}
	}
}

func TestAuthenticatedCommandRunsTheTargetInThePasswordReceivingSudo(t *testing.T) {
	t.Parallel()

	command := buildAuthenticatedCommand(types.Command{
		Exec:     "dscl",
		Args:     []string{".", "-create", "/Users/levi", "UserShell", "/opt/homebrew/bin/fish"},
		Elevated: true,
	})
	want := []string{"sudo", "-S", "-p", "", "--", "dscl", ".", "-create", "/Users/levi", "UserShell", "/opt/homebrew/bin/fish"}
	if !slices.Equal(command.Args, want) {
		t.Fatalf("authenticated argv = %q, want %q", command.Args, want)
	}
}

func TestPasswordInputKeepsSecretOutOfArgvAndPreservesCommandInput(t *testing.T) {
	t.Parallel()

	password := []byte("correct horse battery staple")
	input := passwordInput(password, "target input\n")
	if got, want := string(input), "correct horse battery staple\ntarget input\n"; got != want {
		t.Fatalf("stdin = %q, want %q", got, want)
	}
	command := buildAuthenticatedCommand(types.Command{Exec: "chpasswd", Elevated: true})
	for _, arg := range command.Args {
		if strings.Contains(arg, "correct horse") {
			t.Fatal("password appeared in sudo argv")
		}
	}
}

// A cold cache is reported as cold, and does not prompt on the way there.
func TestSudoCredentialsCachedWhenTheProbeFails(t *testing.T) {
	defer SetSudoProbeForTest(func(context.Context) error { return errors.New("a password is required") })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if sudoCredentialsCached() {
		t.Error("a failing probe was read as a warm cache")
	}
}

// A warm cache is reported as warm, so the command runs captured and the
// dashboard is not torn down for nothing.
func TestSudoCredentialsCachedWhenTheProbeSucceeds(t *testing.T) {
	defer SetSudoProbeForTest(func(context.Context) error { return nil })()
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
	defer SetSudoProbeForTest(func(context.Context) error {
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

// A cold result is throttled as cold, not converted to warm. Every caller still
// hands its command the terminal, but a stalled policy backend is not retried
// once per package.
func TestSudoProbeIsThrottledWhileCold(t *testing.T) {
	var calls int
	defer SetSudoProbeForTest(func(context.Context) error {
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
	if calls != 1 {
		t.Errorf("probe ran %d times for 3 cold calls, want 1", calls)
	}
}

// A cold probe authenticates once through the controlled password path. The
// actual commands remain captured, so raw package and user-management output
// cannot collide with the dashboard.
func TestColdProbeAuthenticatesOnceAndKeepsCommandsCaptured(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	defer SetSudoProbeForTest(func(context.Context) error {
		return errors.New("a password is required")
	})()
	var authentications int
	defer SetSudoPasswordForTest(func() ([]byte, error) {
		authentications++
		return []byte("test password"), nil
	})()
	defer SetSudoValidateForTest(func(context.Context, []byte) error { return nil })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	rec := &recordingReporter{}
	defer reporting.Set(rec)()
	for range 2 {
		if err := RunCommand(types.Command{Exec: "true", Escalates: true}, false); err != nil {
			t.Fatalf("RunCommand: %v", err)
		}
	}

	var handovers int
	for _, event := range rec.events {
		if _, ok := event.(reporting.TerminalReq); ok {
			handovers++
		}
	}
	if authentications != 1 {
		t.Errorf("authentications = %d, want one", authentications)
	}
	if handovers != 0 {
		t.Errorf("command terminal handovers = %d, want none", handovers)
	}
}

func TestPromptedPasswordIsReusedBySeparateManagedSudo(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	defer SetSudoProbeForTest(func(context.Context) error { return errors.New("cold") })()
	var prompts int
	defer SetSudoPasswordForTest(func() ([]byte, error) {
		prompts++
		return []byte("test password"), nil
	})()
	defer SetSudoValidateForTest(func(context.Context, []byte) error { return nil })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if err := ensureSudoCredentials(); err != nil {
		t.Fatalf("warming for self-escalating command: %v", err)
	}
	command, input, err := commandForRun(types.Command{Exec: "dscl", Elevated: true})
	defer zeroBytes(input)
	if err != nil {
		t.Fatalf("commandForRun: %v", err)
	}
	if prompts != 1 {
		t.Fatalf("password prompts = %d, want the run-scoped password reused", prompts)
	}
	if !slices.Equal(command.Args, []string{"sudo", "-S", "-p", "", "--", "dscl"}) {
		t.Fatalf("managed command argv = %q, want password-bound sudo", command.Args)
	}
}

func TestRunLifecycleErasesCachedSudoPassword(t *testing.T) {
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	cacheSudoPassword([]byte("temporary secret"))
	release := BeginRun()
	if password := cachedSudoPassword(); len(password) != 0 {
		zeroBytes(password)
		t.Fatal("BeginRun retained a password from an earlier run")
	}

	cacheSudoPassword([]byte("current run secret"))
	release()
	if password := cachedSudoPassword(); len(password) != 0 {
		zeroBytes(password)
		t.Fatal("run release retained the current run password")
	}
}

func TestSudoAuthenticationFailureStopsTheCommand(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	defer SetSudoProbeForTest(func(context.Context) error { return errors.New("cold") })()
	defer SetSudoPasswordForTest(func() ([]byte, error) { return nil, errors.New("denied") })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	err := RunCommand(types.Command{Exec: "true", Elevated: true}, false)
	if !errors.Is(err, ErrSudoAuthentication) {
		t.Fatalf("RunCommand error = %v, want ErrSudoAuthentication", err)
	}
}

func TestNoTerminalLeavesCommandScopedNopasswdPathAvailable(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("no sudo on windows")
	}

	defer SetSudoProbeForTest(func(context.Context) error { return errors.New("cold") })()
	defer SetSudoPasswordForTest(func() ([]byte, error) { return nil, errNoSudoTerminal })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if err := runCommand(types.Command{Exec: "true", Escalates: true}, false); err != nil {
		t.Fatalf("headless command was rejected before its own policy could run: %v", err)
	}
}

// Cancellation is stronger than the fallback deadline: ctrl-c must stop a
// blocked policy lookup now, not leave the run apparently frozen for ten more
// seconds before it can return ErrCancelled.
func TestSudoProbeStopsWhenTheRunIsCancelled(t *testing.T) {
	started := make(chan struct{})
	defer SetSudoProbeForTest(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	release := BeginRun()
	defer release()

	result := make(chan bool, 1)
	go func() { result <- sudoCredentialsCached() }()
	<-started
	Cancel()

	select {
	case warm := <-result:
		if warm {
			t.Fatal("a cancelled probe was read as a warm cache")
		}
	case <-time.After(time.Second):
		t.Fatal("sudo probe ignored run cancellation")
	}
}

// A probe that never returns must not take the run with it.
//
// -n means sudo will not prompt, but that is not the same as sudo returning:
// a policy lookup can block underneath it, and the probe holds
// sudoValidateMu while it runs. Unbounded, a stall there is not one slow
// command but every command after it, waiting on a mutex nobody releases.
func TestSudoProbeIsBounded(t *testing.T) {
	t.Parallel()

	if sudoProbeTimeout <= 0 {
		t.Fatal("the sudo probe has no deadline")
	}
	if sudoProbeTimeout > 30*time.Second {
		t.Errorf("sudoProbeTimeout = %v, too long to hold every later command behind", sudoProbeTimeout)
	}
}

// A probe that timed out is a cold cache, not a warm one. Reading a deadline
// as success would send the command down the captured path, straight back to
// the invisible prompt this all exists to avoid.
func TestSudoTimeoutReadsAsColdCache(t *testing.T) {
	defer SetSudoProbeForTest(func(context.Context) error { return context.DeadlineExceeded })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	if sudoCredentialsCached() {
		t.Error("a timed-out probe was read as a warm cache")
	}
}

// The lock is not held past the probe: a slow one delays its own caller and
// nobody else's next attempt.
func TestSudoProbeDoesNotHoldTheLockAfterReturning(t *testing.T) {
	defer SetSudoProbeForTest(func(context.Context) error { return nil })()
	resetSudoThrottleForTest()
	defer resetSudoThrottleForTest()

	sudoCredentialsCached()

	done := make(chan struct{})
	go func() {
		resetSudoThrottleForTest()
		sudoCredentialsCached()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sudoValidateMu was still held after the probe returned")
	}
}
