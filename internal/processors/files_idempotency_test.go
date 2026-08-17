package processors

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
)

func filesTestConfig(blueprintDir string) *types.InitConfig {
	return &types.InitConfig{
		Init: types.Init{Location: blueprintDir, Format: "yaml"},
		Variables: types.Variables{
			User:        types.UserInfo{Username: "testuser", Home: "/home/testuser"},
			UserDefined: map[string]interface{}{},
		},
	}
}

func runFilesBlueprint(t *testing.T, blueprintDir, blueprint string) error {
	t.Helper()
	return ProcessFiles([]byte(blueprint), blueprintDir, "yaml", &types.OSInfo{}, filesTestConfig(blueprintDir))
}

// Every action has to converge: `rwr all` is run repeatedly against the same
// machine, and one file that errors because its work was already done aborts the
// entire run.

func TestSymlink_RerunIsANoOp(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	if err := os.MkdirAll(filepath.Join(blueprintDir, "dotfiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blueprintDir, "dotfiles", "vimrc"), []byte("set nocp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "home", ".vimrc")
	blueprint := "files:\n  - name: vimrc\n    action: symlink\n    source: dotfiles\n    target: " + link + "\n"

	for i := 0; i < 2; i++ {
		if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
			t.Fatalf("run %d failed: %v", i+1, err)
		}
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("expected a symlink at %s: %v", link, err)
	}
	if want := filepath.Join(blueprintDir, "dotfiles", "vimrc"); got != want {
		t.Errorf("symlink points at %s, want %s", got, want)
	}
}

func TestSymlink_PointingElsewhereIsReplaced(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	if err := os.MkdirAll(filepath.Join(blueprintDir, "dotfiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blueprintDir, "dotfiles", "vimrc"), []byte("set nocp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".vimrc")
	stale := filepath.Join(root, "stale")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(stale, link); err != nil {
		t.Fatal(err)
	}

	blueprint := "files:\n  - name: vimrc\n    action: symlink\n    source: dotfiles\n    target: " + link + "\n"
	if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(blueprintDir, "dotfiles", "vimrc"); got != want {
		t.Errorf("symlink points at %s, want %s", got, want)
	}
}

func TestSymlink_RegularFileInTheWayIsAnError(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	if err := os.MkdirAll(filepath.Join(blueprintDir, "dotfiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blueprintDir, "dotfiles", "vimrc"), []byte("set nocp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, ".vimrc")
	if err := os.WriteFile(link, []byte("hand written"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The occupied target is a ledger failure, not an abort: the processor
	// returns nil so later items still run, and the failure reaches the exit
	// code through All().
	resetFailures()
	defer resetFailures()

	blueprint := "files:\n  - name: vimrc\n    action: symlink\n    source: dotfiles\n    target: " + link + "\n"
	if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
		t.Fatalf("runFilesBlueprint = %v, want nil: item failures belong in the ledger", err)
	}
	err := failureError()
	if err == nil {
		t.Fatal("expected a recorded failure when a regular file occupies the symlink target")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("failure should say what is in the way, got: %v", err)
	}

	content, readErr := os.ReadFile(link)
	if readErr != nil || string(content) != "hand written" {
		t.Errorf("the existing file must be left alone, got %q (%v)", content, readErr)
	}
}

func TestDelete_RerunIsANoOp(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	target := filepath.Join(root, "obsolete.conf")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	blueprint := "files:\n  - name: obsolete.conf\n    action: delete\n    source: unused\n    target: " + root + "/\n"

	for i := 0; i < 2; i++ {
		if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
			t.Fatalf("run %d failed: %v", i+1, err)
		}
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("expected %s to be gone, got %v", target, err)
	}
}

func TestCreate_RerunIsANoOpAndKeepsTheMode(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	blueprint := "files:\n  - name: app.conf\n    action: create\n    content: \"key = value\"\n    mode: 0600\n    target: " + root + "/\n"

	for i := 0; i < 2; i++ {
		if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
			t.Fatalf("run %d failed: %v", i+1, err)
		}
	}

	target := filepath.Join(root, "app.conf")
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode is %o, want 600", perm)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "key = value" {
		t.Errorf("content is %q, want the content written once, not twice", content)
	}
}

// A template can render a .netrc or a gh config - that is what exposing
// credentials to templates is for - so a rendered template with no declared mode
// must not be created world-readable and narrowed afterwards.

func TestTemplate_DefaultsToPrivateMode(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	templatesDir := filepath.Join(blueprintDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "netrc"), []byte("password {{ .User.username }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	blueprint := "templates:\n  - name: netrc\n    action: create\n    source: templates\n    target: " + root + "/\n"
	if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(root, "netrc"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("rendered template mode is %o, want 600", perm)
	}
}

func TestTemplate_ExplicitModeWins(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	templatesDir := filepath.Join(blueprintDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "motd"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	blueprint := "templates:\n  - name: motd\n    action: create\n    source: templates\n    mode: 0644\n    target: " + root + "/\n"
	if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(root, "motd"))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mode is %o, want 644", perm)
	}
}

// The mode has to be narrowed before the secret is written, not after: a chmod
// that follows the write leaves a window in which the credential is readable by
// everyone. Rewriting a file an earlier, buggier run left at 0644 is the case
// where that window is observable, so watch for it.
func TestCreate_SecretIsNeverObservableAt0644(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	templatesDir := filepath.Join(blueprintDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	secret := strings.Repeat("password abcdefghijklmnop\n", 400_000)
	if err := os.WriteFile(filepath.Join(templatesDir, "netrc"), []byte(secret), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(root, "netrc")
	if err := os.WriteFile(target, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	var exposed atomic.Bool
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if info, err := os.Stat(target); err == nil {
				if info.Size() > int64(len("stale")) && info.Mode().Perm()&0o077 != 0 {
					exposed.Store(true)
				}
			}
			time.Sleep(time.Microsecond)
		}
	}()

	blueprint := "templates:\n  - name: netrc\n    action: create\n    source: templates\n    target: " + root + "/\n"
	err := runFilesBlueprint(t, blueprintDir, blueprint)
	close(done)
	if err != nil {
		t.Fatal(err)
	}

	if exposed.Load() {
		t.Error("the rendered secret was group- or world-readable while being written")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("final mode is %o, want 600", perm)
	}
}

// Every action resolves its target through one helper, so `target:` means the
// same path whichever action reads it, and the dry-run line names the path the
// run would actually touch.

func TestTargetPath_SameForCreateAndCopy(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	sourceDir := filepath.Join(blueprintDir, "dotfiles")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bashrc"), []byte("copied\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No trailing separator and nothing at the path: the target is the file itself.
	rename := filepath.Join(root, ".bashrc")

	create := "files:\n  - name: bashrc\n    action: create\n    content: \"created\"\n    target: " + rename + "\n"
	if err := runFilesBlueprint(t, blueprintDir, create); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(rename)
	if err != nil {
		t.Fatalf("create should have written %s: %v", rename, err)
	}
	if string(content) != "created" {
		t.Errorf("content is %q", content)
	}
	if _, err := os.Stat(filepath.Join(rename, "bashrc")); err == nil {
		t.Errorf("create wrote %s as a directory", rename)
	}

	copyBlueprint := "files:\n  - name: bashrc\n    action: copy\n    source: dotfiles\n    target: " + rename + "\n"
	if err := runFilesBlueprint(t, blueprintDir, copyBlueprint); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(rename)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "copied\n" {
		t.Errorf("copy wrote %q to a different path than create did", content)
	}
}

func TestTargetPath_TrailingSeparatorMeansDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "config")

	if got, want := resolveTargetPath(nested+"/", "app.conf"), filepath.Join(nested, "app.conf"); got != want {
		t.Errorf("trailing separator: got %s, want %s", got, want)
	}
	if got, want := resolveTargetPath(nested, "app.conf"), nested; got != want {
		t.Errorf("no trailing separator, nothing at the path: got %s, want %s", got, want)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := resolveTargetPath(nested, "app.conf"), filepath.Join(nested, "app.conf"); got != want {
		t.Errorf("no trailing separator, existing directory: got %s, want %s", got, want)
	}
}

func TestDryRunPath_MatchesWhereCreateWrites(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	file := types.File{Name: "app.conf", Action: "create", Content: "x", Target: filepath.Join(root, "etc") + "/"}

	_, dryRunPath, err := determineSourceAndTargetPaths(file, blueprintDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := createFile(file, dryRunPath, &types.OSInfo{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dryRunPath); err != nil {
		t.Errorf("the dry-run path is not where create wrote: %v", err)
	}
}

// A `templates:` entry that imports another file has to pull that file's
// templates. Reading its `files:` list instead drops every imported template
// without an error.
func TestTemplateImports_LoadTemplatesNotFiles(t *testing.T) {
	root := t.TempDir()
	blueprintDir := filepath.Join(root, "blueprints")
	templatesDir := filepath.Join(blueprintDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "gitconfig"), []byte("[user]\n\tname = imported\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	imported := "templates:\n  - name: gitconfig\n    action: create\n    source: templates\n    target: " + root + "/\n"
	if err := os.WriteFile(filepath.Join(blueprintDir, "common.yaml"), []byte(imported), 0o644); err != nil {
		t.Fatal(err)
	}

	blueprint := "templates:\n  - import: common.yaml\n"
	if err := runFilesBlueprint(t, blueprintDir, blueprint); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(root, "gitconfig"))
	if err != nil {
		t.Fatalf("the imported template was dropped: %v", err)
	}
	if !strings.Contains(string(content), "name = imported") {
		t.Errorf("rendered content is %q", content)
	}
}
