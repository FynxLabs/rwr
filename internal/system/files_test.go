package system

import (
	"os"
	"path/filepath"
	"testing"
)

// Commands are argv now, so no shell expands a leading ~ on the way to a
// program. Every blueprint path that reaches a command or a filesystem call has
// to be expanded here instead, or "~/.ssh" creates a directory literally named
// "~" in the working directory.
func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}

	tests := []struct {
		in   string
		want string
	}{
		{"~/.ssh", filepath.Join(home, ".ssh")},
		{"~/.ssh/id_ed25519", filepath.Join(home, ".ssh", "id_ed25519")},
		{"/etc/ssh", "/etc/ssh"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := ExpandPath(tt.in); got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
