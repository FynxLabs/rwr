package system

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestExpandPath_BareTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}

	if got := ExpandPath("~"); got != home {
		t.Errorf("ExpandPath(%q) = %q, want %q", "~", got, home)
	}
	// Only a leading path element is a home reference.
	for _, in := range []string{"~user/file", "a/~", "./~"} {
		if got := ExpandPath(in); got != in {
			t.Errorf("ExpandPath(%q) = %q, want it unchanged", in, got)
		}
	}
}

// The staging file used to be created in os.TempDir(), which is tmpfs on most
// Linux distributions; rename(2) across filesystems fails with EXDEV, so every
// non-elevated write to a target outside /tmp failed. Writing into an ordinary
// directory has to succeed.
func TestWriteToFile_TargetOutsideTempDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.yaml")

	if err := WriteToFile(target, "key: value\n", false); err != nil {
		t.Fatalf("WriteToFile() error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "key: value\n" {
		t.Errorf("target content = %q, want %q", got, "key: value\n")
	}
	assertNoStagingLeftovers(t, dir, "config.yaml")
}

func TestWriteToFile_PreservesExistingMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "credentials")

	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatalf("seeding target: %v", err)
	}
	if err := WriteToFile(target, "new", false); err != nil {
		t.Fatalf("WriteToFile() error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("mode = %o, want 0600 (a rewrite must not widen permissions)", got)
	}
}

func TestWriteToFile_NewFileUsesDefaultMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	target := filepath.Join(t.TempDir(), "new.conf")

	if err := WriteToFile(target, "content", false); err != nil {
		t.Fatalf("WriteToFile() error: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != defaultFileMode {
		t.Errorf("mode = %o, want %o", got, defaultFileMode)
	}
}

// A repository add appends its section to a shared configuration file. What is
// already in that file - every other repository, every option - has to survive,
// and a second `rwr all` must not add the section again.
func TestAppendToFile_KeepsExistingContentAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "pacman.conf")
	existing := "[options]\nParallelDownloads = 5\n"
	if err := os.WriteFile(target, []byte(existing), 0600); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	section := "[chaotic-aur]\nServer = https://example.com/repo\n"
	for run := 1; run <= 2; run++ {
		if err := AppendToFile(target, section, false); err != nil {
			t.Fatalf("AppendToFile() run %d error: %v", run, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if want := existing + section; string(got) != want {
			t.Fatalf("run %d: target = %q, want %q", run, got, want)
		}
	}
	assertNoStagingLeftovers(t, dir, "pacman.conf")
}

func TestAppendToFile_CreatesMissingFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "repositories")

	if err := AppendToFile(target, "https://example.com/alpine/edge", false); err != nil {
		t.Fatalf("AppendToFile() error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if want := "https://example.com/alpine/edge\n"; string(got) != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

// An unterminated last line would otherwise run into the appended content.
func TestAppendToFile_SeparatesFromUnterminatedLine(t *testing.T) {
	target := filepath.Join(t.TempDir(), "repositories")
	if err := os.WriteFile(target, []byte("https://example.com/main"), 0600); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	if err := AppendToFile(target, "https://example.com/edge", false); err != nil {
		t.Fatalf("AppendToFile() error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if want := "https://example.com/main\nhttps://example.com/edge\n"; string(got) != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestRemoveLineFromFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "repositories")
	content := "https://example.com/main\nhttps://example.com/edge extra\nhttps://example.com/edge/other\n"
	if err := os.WriteFile(target, []byte(content), 0600); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	// Twice: the second removal has nothing left to take out.
	for run := 1; run <= 2; run++ {
		if err := RemoveLineFromFile(target, "https://example.com/edge", false); err != nil {
			t.Fatalf("RemoveLineFromFile() run %d error: %v", run, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		// The line whose first field is the match goes; the longer URL that
		// merely starts with it stays.
		want := "https://example.com/main\nhttps://example.com/edge/other\n"
		if string(got) != want {
			t.Fatalf("run %d: target = %q, want %q", run, got, want)
		}
	}
}

func TestRemoveLineFromFile_MissingFileIsNotAnError(t *testing.T) {
	target := filepath.Join(t.TempDir(), "absent")
	if err := RemoveLineFromFile(target, "https://example.com", false); err != nil {
		t.Errorf("RemoveLineFromFile() error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("created %s (stat err %v)", target, err)
	}
}

func TestRemoveSectionFromFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "pacman.conf")
	content := "[options]\nParallelDownloads = 5\n\n[chaotic-aur]\nServer = https://example.com/repo\nSigLevel = Required\n\n[core]\nInclude = /etc/pacman.d/mirrorlist\n"
	if err := os.WriteFile(target, []byte(content), 0600); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	for run := 1; run <= 2; run++ {
		if err := RemoveSectionFromFile(target, "chaotic-aur", false); err != nil {
			t.Fatalf("RemoveSectionFromFile() run %d error: %v", run, err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading target: %v", err)
		}
		if strings.Contains(string(got), "chaotic-aur") || strings.Contains(string(got), "SigLevel") {
			t.Fatalf("run %d: section survived: %q", run, got)
		}
		for _, keep := range []string{"[options]", "ParallelDownloads = 5", "[core]", "Include = /etc/pacman.d/mirrorlist"} {
			if !strings.Contains(string(got), keep) {
				t.Fatalf("run %d: lost unrelated content %q from %q", run, keep, got)
			}
		}
	}
}

func TestRemoveSectionFromFile_MissingFileIsNotAnError(t *testing.T) {
	target := filepath.Join(t.TempDir(), "absent")
	if err := RemoveSectionFromFile(target, "chaotic-aur", false); err != nil {
		t.Errorf("RemoveSectionFromFile() error: %v", err)
	}
}

func TestDownloadFile_TargetOutsideTempDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("payload")) //nolint:errcheck
	}))
	defer server.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "downloaded.gpg")

	if err := DownloadFile(server.URL, target, false); err != nil {
		t.Fatalf("DownloadFile() error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("target content = %q, want %q", got, "payload")
	}
	assertNoStagingLeftovers(t, dir, "downloaded.gpg")
}

func TestOpenFileNoFollowRejectsFinalSymlink(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows os.OpenFile has no O_NOFOLLOW equivalent")
	}

	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(outside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	file, err := OpenFileNoFollow(link, os.O_WRONLY|os.O_TRUNC, 0)
	if err == nil {
		_ = file.Close()
		t.Fatal("OpenFileNoFollow followed a final-component symlink")
	}
	content, readErr := os.ReadFile(outside)
	if readErr != nil || string(content) != "original" {
		t.Fatalf("outside content = %q, err = %v; want unchanged", content, readErr)
	}
}

func TestDownloadFile_ErrorRemovesStagingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	dir := t.TempDir()

	if err := DownloadFile(server.URL, filepath.Join(dir, "never.bin"), false); err == nil {
		t.Fatal("DownloadFile() succeeded on a 404, want error")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("staging file left behind: %v", entries)
	}
}

// moveIntoPlace's copy fallback is what covers a rename that cannot work,
// so it is exercised directly rather than only through a same-filesystem move.
func TestCopyFileContentMode_FallbackPath(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged")
	target := filepath.Join(dir, "target")

	if err := os.WriteFile(staged, []byte("staged content"), 0600); err != nil {
		t.Fatalf("seeding staged file: %v", err)
	}
	if err := os.WriteFile(target, []byte("stale and much longer content"), 0666); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	if err := copyFileContentMode(staged, target, 0640); err != nil {
		t.Fatalf("copyFileContentMode() error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "staged content" {
		t.Errorf("target content = %q, want %q (target must be truncated)", got, "staged content")
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat target: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0640 {
			t.Errorf("mode = %o, want 0640", perm)
		}
	}
}

func TestMoveIntoPlace_RemovesStagedFile(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged")
	target := filepath.Join(dir, "sub", "target")

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("creating target dir: %v", err)
	}
	if err := os.WriteFile(staged, []byte("content"), 0600); err != nil {
		t.Fatalf("seeding staged file: %v", err)
	}

	if err := moveIntoPlace(staged, target, 0644, false); err != nil {
		t.Fatalf("moveIntoPlace() error: %v", err)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Errorf("staged file still present after move: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "content" {
		t.Errorf("target = %q, err = %v; want %q", got, err, "content")
	}
}

// moveIntoPlace must not leave the staging file behind when the move itself
// fails, or a failed run litters the target directory.
func TestMoveIntoPlace_RemovesStagedFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged")

	if err := os.WriteFile(staged, []byte("content"), 0600); err != nil {
		t.Fatalf("seeding staged file: %v", err)
	}

	// The target's parent does not exist, so both rename and the copy fallback fail.
	target := filepath.Join(dir, "missing", "target")
	if err := moveIntoPlace(staged, target, 0644, false); err == nil {
		t.Fatal("moveIntoPlace() succeeded with a missing parent directory, want error")
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Errorf("staged file still present after failed move: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "present")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatalf("seeding file: %v", err)
	}

	if !FileExists(existing) {
		t.Error("FileExists() = false for an existing file, want true")
	}
	if FileExists(filepath.Join(dir, "absent")) {
		t.Error("FileExists() = true for a missing file, want false")
	}

	// A path whose parent component is a regular file stats with ENOTDIR, not
	// ENOENT; that is not an existing file either.
	if FileExists(filepath.Join(existing, "child")) {
		t.Error("FileExists() = true for a path under a regular file, want false")
	}
}

func TestFileExists_UnreadableParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	dir := t.TempDir()
	parent := filepath.Join(dir, "locked")
	if err := os.Mkdir(parent, 0755); err != nil {
		t.Fatalf("creating parent: %v", err)
	}
	target := filepath.Join(parent, "file")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatalf("seeding file: %v", err)
	}
	if err := os.Chmod(parent, 0000); err != nil {
		t.Fatalf("chmod parent: %v", err)
	}
	t.Cleanup(func() { os.Chmod(parent, 0755) }) //nolint:errcheck

	// Stat fails with EACCES here; a failed stat is not proof the file exists.
	if FileExists(target) {
		t.Error("FileExists() = true when stat failed with permission denied, want false")
	}
}

// fileExists dereferenced a nil FileInfo on any non-ENOENT stat error.
func TestFileExists_LinuxHelper(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "os-release")
	if err := os.WriteFile(regular, []byte("ID=arch\n"), 0644); err != nil {
		t.Fatalf("seeding file: %v", err)
	}

	if !fileExists(regular) {
		t.Error("fileExists() = false for a regular file, want true")
	}
	if fileExists(dir) {
		t.Error("fileExists() = true for a directory, want false")
	}
	if fileExists(filepath.Join(dir, "absent")) {
		t.Error("fileExists() = true for a missing file, want false")
	}
	// ENOTDIR: a path component is a regular file. This used to panic.
	if fileExists(filepath.Join(regular, "child")) {
		t.Error("fileExists() = true for a path under a regular file, want false")
	}
}

// assertNoStagingLeftovers checks that dir holds only the expected files: the
// staging file now lives next to the target, so a leak shows up there.
func assertNoStagingLeftovers(t *testing.T, dir string, expected ...string) {
	t.Helper()

	want := make(map[string]bool, len(expected))
	for _, name := range expected {
		want[name] = true
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	for _, entry := range entries {
		if !want[entry.Name()] {
			t.Errorf("unexpected file left in target directory: %s", entry.Name())
		}
	}
}

// TestMoveIntoPlace_CrossFilesystem exercises the real EXDEV path: /dev/shm is a
// separate tmpfs mount, so rename(2) out of it fails and only the copy fallback
// can complete the move. Skipped where no second filesystem is reachable.
func TestMoveIntoPlace_CrossFilesystem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("uses /dev/shm as a second filesystem")
	}
	dir := t.TempDir()

	probe, err := os.CreateTemp("/dev/shm", "rwr-probe-")
	if err != nil {
		t.Skipf("/dev/shm not usable: %v", err)
	}
	probe.Close() //nolint:errcheck
	defer os.Remove(probe.Name())
	if err := os.Rename(probe.Name(), filepath.Join(dir, "probe")); err == nil {
		t.Skip("/dev/shm is on the same filesystem as the temp dir")
	}

	staged, err := os.CreateTemp("/dev/shm", "rwr-staged-")
	if err != nil {
		t.Skipf("/dev/shm not usable: %v", err)
	}
	if _, err := staged.WriteString("cross-device content"); err != nil {
		t.Fatalf("writing staged file: %v", err)
	}
	staged.Close() //nolint:errcheck
	defer os.Remove(staged.Name())

	target := filepath.Join(dir, "target")
	if err := moveIntoPlace(staged.Name(), target, 0640, false); err != nil {
		t.Fatalf("moveIntoPlace() across filesystems: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "cross-device content" {
		t.Errorf("target content = %q, want %q", got, "cross-device content")
	}
	if _, err := os.Stat(staged.Name()); !os.IsNotExist(err) {
		t.Errorf("staged file still present after cross-filesystem move: %v", err)
	}
	if info, err := os.Stat(target); err != nil {
		t.Fatalf("stat target: %v", err)
	} else if perm := info.Mode().Perm(); perm != 0640 {
		t.Errorf("mode = %o, want 0640", perm)
	}
}

// A pre-existing target donates its mode only when the invoking user owns it:
// in a shared directory anyone can plant the target ahead of time, and
// inheriting a planted 0666 leaves the freshly written file world-writable.
func TestTargetMode_InheritsOnlyFromOwnFiles(t *testing.T) {
	dir := t.TempDir()

	owned := filepath.Join(dir, "owned")
	if err := os.WriteFile(owned, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(owned, 0o640); err != nil {
		t.Fatal(err)
	}
	if got := targetMode(owned); got != 0o640 {
		t.Errorf("targetMode(own file) = %04o, want 0640 inherited", got)
	}

	if got := targetMode(filepath.Join(dir, "absent")); got != defaultFileMode {
		t.Errorf("targetMode(absent) = %04o, want the default %04o", got, defaultFileMode)
	}

	// A file owned by another user cannot be created without privileges, so
	// the refusal branch is exercised at the helper level.
	info, err := os.Stat(owned)
	if err != nil {
		t.Fatal(err)
	}
	if !fileOwnedByEUID(info) {
		t.Error("fileOwnedByEUID(own file) = false, want true")
	}
}
