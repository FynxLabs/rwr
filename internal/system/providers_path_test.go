package system

import (
	"os"
	"path/filepath"
	"testing"
)

// With HOME unset, the user-config candidate must be dropped, not degrade to
// the relative path ".config/rwr/providers" — which os.Stat would resolve
// against the CWD, honouring a providers directory planted in whatever
// directory rwr is run from (a cloned repo, /tmp). GetProvidersPath's own doc
// comment rules the CWD out of the search.
func TestGetProvidersPath_UnsetHomeNeverSearchesCWD(t *testing.T) {
	// A CWD that contains the path the buggy join produces.
	cwd := t.TempDir()
	planted := filepath.Join(cwd, ".config/rwr/providers")
	if err := os.MkdirAll(planted, 0o755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	}()

	// t.Setenv restores these afterwards. Empty HOME makes os.UserHomeDir fail
	// on unix; USERPROFILE covers windows.
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	got, err := GetProvidersPath()
	if err != nil {
		// No providers directory found at all is the correct outcome when the
		// only candidate was the planted CWD one.
		return
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("GetProvidersPath returned a relative path %q — resolved against the CWD", got)
	}
	if got == planted || got == ".config/rwr/providers" {
		t.Fatalf("GetProvidersPath honoured a providers directory under the CWD: %q", got)
	}
}
