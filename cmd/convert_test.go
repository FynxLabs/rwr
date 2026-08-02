package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --migrate moves the removed init-file inline sections into blueprint files;
// dry-run touches nothing, --write moves them and the init file stops
// declaring the key.
func TestConvert_MigratesInitInlineSections(t *testing.T) {
	dir := t.TempDir()
	init := "blueprints:\n  format: yaml\n  location: .\npackages:\n  - name: git\n    action: install\n"
	if err := os.WriteFile(filepath.Join(dir, "init.yaml"), []byte(init), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newConvertCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--migrate", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !strings.Contains(out.String(), `move "packages"`) {
		t.Fatalf("dry run did not describe the move:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "packages", "from-init.yaml")); err == nil {
		t.Fatal("dry run wrote files")
	}

	cmd = newConvertCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--migrate", "--write", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("write: %v", err)
	}

	moved, err := os.ReadFile(filepath.Join(dir, "packages", "from-init.yaml"))
	if err != nil || !strings.Contains(string(moved), "git") {
		t.Fatalf("moved blueprint = %q (%v), want the packages entry", moved, err)
	}
	rewritten, err := os.ReadFile(filepath.Join(dir, "init.yaml"))
	if err != nil || strings.Contains(string(rewritten), "packages:") {
		t.Fatalf("init still declares packages (%v):\n%s", err, rewritten)
	}
}

// --to converts every blueprint file, replacing the originals only under
// --write, and warns about comments it cannot carry across.
func TestConvert_ToFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "packages.yaml"), []byte("# tools\npackages:\n  - name: git\n    action: install\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newConvertCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--to", "toml", "--write", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("convert: %v", err)
	}

	if !strings.Contains(out.String(), "NOT preserved") {
		t.Errorf("comment warning missing:\n%s", out.String())
	}
	converted, err := os.ReadFile(filepath.Join(dir, "packages.toml"))
	if err != nil || !strings.Contains(string(converted), `name = "git"`) {
		t.Fatalf("converted = %q (%v)", converted, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "packages.yaml")); err == nil {
		t.Error("original left behind after --write")
	}
}
