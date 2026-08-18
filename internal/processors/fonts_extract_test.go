package processors

import (
	"archive/tar"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/ulikunitz/xz"
)

type tarEntry struct {
	name     string
	typeflag byte
	body     string
	linkname string
}

// writeFontTarball builds a .tar.xz in the shape extractFontTarball expects, so a
// test can hand it entry names a real archive would never contain.
func writeFontTarball(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating tarball: %v", err)
	}
	defer f.Close() //nolint:errcheck

	xzw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatalf("creating xz writer: %v", err)
	}
	tw := tar.NewWriter(xzw)

	for _, e := range entries {
		typeflag := e.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: typeflag,
			Linkname: e.linkname,
			Mode:     0o644,
			Size:     int64(len(e.body)),
		}
		if typeflag != tar.TypeReg {
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("writing tar header %q: %v", e.name, err)
		}
		if hdr.Size > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("writing tar body %q: %v", e.name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := xzw.Close(); err != nil {
		t.Fatalf("closing xz writer: %v", err)
	}
}

func fontOSInfo() *types.OSInfo {
	osInfo := &types.OSInfo{}
	osInfo.System.OS = "linux"
	return osInfo
}

// Entry names in a downloaded archive are attacker-controlled: joining them onto
// the font directory without a containment check lets a member escape it, and for a
// system-scoped install that write happens as root.
func TestExtractFontTarball_RejectsEntriesEscapingDestDir(t *testing.T) {
	tests := []struct {
		name  string
		entry string
	}{
		{"parent traversal", "../../evil.ttf"},
		{"single parent", "../evil.ttf"},
		{"traversal below a subdirectory", "fonts/../../../evil.ttf"},
		{"absolute path", "/tmp/rwr-evil.ttf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := exectest.New()
			defer system.SetExecutor(rec)()

			root := t.TempDir()
			destDir := filepath.Join(root, "nested", "fonts")
			if err := os.MkdirAll(destDir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			tarballPath := filepath.Join(root, "font.tar.xz")
			writeFontTarball(t, tarballPath, []tarEntry{{name: tt.entry, body: "ttf-bytes"}})

			_, err := extractFontTarball(tarballPath, destDir, false, fontOSInfo())
			if err == nil {
				t.Fatal("expected extraction to fail, got nil")
			}

			// Nothing may have been written anywhere under the temp root, and in
			// particular not outside destDir.
			assertNoFileOutside(t, root, destDir, "evil.ttf")
			if _, statErr := os.Stat("/tmp/rwr-evil.ttf"); statErr == nil {
				t.Fatal("absolute entry escaped to /tmp/rwr-evil.ttf")
			}
		})
	}
}

// Links can point anywhere; installing a font never needs one, so they are dropped
// rather than followed.
func TestExtractFontTarball_SkipsLinkEntries(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	root := t.TempDir()
	destDir := filepath.Join(root, "fonts")
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tarballPath := filepath.Join(root, "font.tar.xz")
	writeFontTarball(t, tarballPath, []tarEntry{
		{name: "escape.ttf", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
		{name: "hard.ttf", typeflag: tar.TypeLink, linkname: "../../outside.ttf"},
	})

	if _, err := extractFontTarball(tarballPath, destDir, false, fontOSInfo()); err != nil {
		t.Fatalf("extractFontTarball: %v", err)
	}

	entries, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("reading destDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("destDir is not empty: %v", entries)
	}
}

// A user-scoped install writes under $HOME and must not ask for privilege; only
// location "system" may.
func TestExtractFontTarball_ElevatesOnlyWhenAsked(t *testing.T) {
	tests := []struct {
		name     string
		elevated bool
	}{
		{"user scoped", false},
		{"system scoped", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := exectest.New()
			defer system.SetExecutor(rec)()

			root := t.TempDir()
			destDir := filepath.Join(root, "fonts")
			if err := os.MkdirAll(destDir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			tarballPath := filepath.Join(root, "font.tar.xz")
			writeFontTarball(t, tarballPath, []tarEntry{{name: "Good.ttf", body: "ttf-bytes"}})

			if _, err := extractFontTarball(tarballPath, destDir, tt.elevated, fontOSInfo()); err != nil {
				t.Fatalf("extractFontTarball: %v", err)
			}

			// system.CopyFile shells out only for the elevated copy, so a recorded
			// call is the observable difference between the two scopes.
			if tt.elevated && len(rec.Calls) == 0 {
				t.Error("system-scoped install did not request elevation")
			}
			if !tt.elevated {
				if len(rec.Calls) != 0 {
					t.Errorf("user-scoped install ran privileged commands: %v", rec.Calls)
				}
				if _, err := os.Stat(filepath.Join(destDir, "Good.ttf")); err != nil {
					t.Errorf("font was not written to destDir: %v", err)
				}
			}
		})
	}
}

func TestResolveTarEntryPath(t *testing.T) {
	destDir := filepath.Join(string(filepath.Separator), "tmp", "rwr-fonts")

	tests := []struct {
		name    string
		entry   string
		want    string
		wantErr bool
	}{
		{name: "plain name", entry: "Good.ttf", want: filepath.Join(destDir, "Good.ttf")},
		{name: "nested name", entry: "sub/Good.ttf", want: filepath.Join(destDir, "sub", "Good.ttf")},
		{name: "dot prefixed", entry: "./Good.ttf", want: filepath.Join(destDir, "Good.ttf")},
		{name: "traversal", entry: "../../evil.ttf", wantErr: true},
		{name: "traversal mid path", entry: "a/../../evil.ttf", wantErr: true},
		{name: "absolute", entry: "/etc/evil.ttf", wantErr: true},
		{name: "empty", entry: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTarEntryPath(destDir, tt.entry)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTarEntryPath: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// font.Name is concatenated into the nerd-fonts download URL, so a name carrying a
// path separator selects a different repository's release.
func TestValidateFontName(t *testing.T) {
	tests := []struct {
		name    string
		font    string
		wantErr bool
	}{
		{name: "plain", font: "FiraCode"},
		{name: "with spaces", font: "Fira Code"},
		{name: "empty", font: "", wantErr: true},
		{name: "slash", font: "../../attacker/repo/releases/download/v1/Evil", wantErr: true},
		{name: "backslash", font: `..\..\Evil`, wantErr: true},
		{name: "dotdot only", font: "..", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFontName(tt.font)
			if tt.wantErr && err == nil {
				t.Error("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// getLatestReleaseURL is a network call; --dry-run has to work offline.
func TestProcessFonts_DryRunMakesNoNetworkCall(t *testing.T) {
	system.SetDryRun(true)
	defer system.SetDryRun(false)

	blueprint := []byte(`
fonts:
  - name: "FiraCode"
    action: "install"
    location: "user"
`)

	if err := ProcessFonts(blueprint, t.TempDir(), "yaml", fontOSInfo(), &types.InitConfig{}); err != nil {
		t.Fatalf("ProcessFonts: %v", err)
	}
}

func assertNoFileOutside(t *testing.T, root, destDir, base string) {
	t.Helper()

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Base(path) != base {
			return nil
		}
		if !strings.HasPrefix(path, destDir+string(filepath.Separator)) {
			t.Errorf("file written outside destDir: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
}

// The face filter used to be ".ttf" alone: an OTF-only archive - several Nerd
// Fonts ship only .otf - "installed successfully" with zero files written.
func TestExtractFontTarball_InstallsEveryFontFace(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	root := t.TempDir()
	destDir := filepath.Join(root, "fonts")
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tarballPath := filepath.Join(root, "font.tar.xz")
	writeFontTarball(t, tarballPath, []tarEntry{
		{name: "Mono-Regular.otf", body: "otf-bytes"},
		{name: "Mono-Bold.TTF", body: "ttf-bytes"},
		{name: "README.md", body: "not a font"},
	})

	extracted, err := extractFontTarball(tarballPath, destDir, false, fontOSInfo())
	if err != nil {
		t.Fatalf("extractFontTarball: %v", err)
	}
	if extracted != 2 {
		t.Errorf("extracted = %d, want 2 (both faces, not the README)", extracted)
	}
	for _, name := range []string{"Mono-Regular.otf", "Mono-Bold.TTF"} {
		info, err := os.Stat(filepath.Join(destDir, name))
		if err != nil {
			t.Errorf("expected %s to be installed: %v", name, err)
		} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0o644 {
			t.Errorf("%s mode = %o, want 0644", name, info.Mode().Perm())
		}
	}
	if _, err := os.Stat(filepath.Join(destDir, "README.md")); err == nil {
		t.Error("README.md installed into the font directory")
	}
}

// Removal has to see the same faces the install writes, or an installed OTF
// face survives its own removal.
func TestRemoveFont_RemovesEveryFontFace(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	home := t.TempDir()
	t.Setenv("HOME", home)
	fontDir := getFontDirectory("user", fontOSInfo())
	if err := os.MkdirAll(fontDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"Mono-Regular.otf", "Mono-Bold.ttf"} {
		if err := os.WriteFile(filepath.Join(fontDir, name), []byte("face"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	if err := removeFont(types.Font{Name: "Mono", Action: "remove", Location: "user"}, fontOSInfo()); err != nil {
		t.Fatalf("removeFont: %v", err)
	}

	entries, err := os.ReadDir(fontDir)
	if err != nil {
		t.Fatalf("reading font dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("font dir not empty after removal: %v", entries)
	}
}
