package system

import (
	"os"
	"path/filepath"
	"testing"
)

// On unix, group/world-writable definitions are skipped; a 0644 one loads.
// On Windows, Go synthesizes mode 0666 for every writable file, so the same
// check rejected every provider definition a Windows user could ever write —
// the platform gets no mode check at all rather than one that always fails.
func TestIsProviderFileTrustedOn(t *testing.T) {
	dir := t.TempDir()

	private := filepath.Join(dir, "private.toml")
	if err := os.WriteFile(private, []byte("[provider]\nname='x'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loose := filepath.Join(dir, "loose.toml")
	if err := os.WriteFile(loose, []byte("[provider]\nname='x'\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	// WriteFile's mode passes through the umask; make the mode explicit.
	if err := os.Chmod(loose, 0o666); err != nil {
		t.Fatal(err)
	}

	// A tight file inside a group/world-writable directory: anyone who can
	// write the directory can replace the file, so it must be skipped too.
	looseDir := filepath.Join(dir, "loosedir")
	if err := os.Mkdir(looseDir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(looseDir, 0o777); err != nil {
		t.Fatal(err)
	}
	tightInLooseDir := filepath.Join(looseDir, "tight.toml")
	if err := os.WriteFile(tightInLooseDir, []byte("[provider]\nname='x'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name string
		goos string
		path string
		want bool
	}{
		{name: "linux private file loads", goos: "linux", path: private, want: true},
		{name: "linux world-writable file skipped", goos: "linux", path: loose, want: false},
		{name: "linux tight file in loose directory skipped", goos: "linux", path: tightInLooseDir, want: false},
		// The same 0666 mode Windows reports for every normal file.
		{name: "windows 0666 file loads", goos: "windows", path: loose, want: true},
		{name: "windows missing file skipped", goos: "windows", path: filepath.Join(dir, "nope.toml"), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProviderFileTrustedOn(tt.goos, tt.path); got != tt.want {
				t.Errorf("isProviderFileTrustedOn(%q, %q) = %v, want %v", tt.goos, tt.path, got, tt.want)
			}
		})
	}
}
