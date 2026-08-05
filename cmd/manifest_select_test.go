package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Headless runs must never prompt: multiple manifest matches without
// --config-name is an error listing the candidates, and --config-name always
// wins without consulting the matchers.
func TestSelectFromManifest_HeadlessSelection(t *testing.T) {
	dir := t.TempDir()
	manifest := "configurations:\n" +
		"  - name: desktop\n    init: Desktop/init.yaml\n    os: " + runtime.GOOS + "\n" +
		"  - name: server\n    init: Server/init.yaml\n    os: " + runtime.GOOS + "\n" +
		"  - name: other\n    init: Other/init.yaml\n    os: never-matches\n"
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewAppConfig()
	app.NoTUI = true // headless: prompting is forbidden

	_, err := selectFromManifest(app, manifestPath)
	if err == nil {
		t.Fatal("multiple matches selected silently in headless mode")
	}
	for _, want := range []string{"desktop", "server", "--config-name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("headless error missing %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "other") {
		t.Errorf("unmatched entry listed as a candidate: %v", err)
	}

	app.ConfigName = "server"
	selected, err := selectFromManifest(app, manifestPath)
	if err != nil {
		t.Fatalf("--config-name selection: %v", err)
	}
	if selected != filepath.Join(dir, "Server", "init.yaml") {
		t.Errorf("selected = %s, want Server/init.yaml under the repo root", selected)
	}
}

// In headless mode a single match still resolves without prompting, so CI and
// pipes keep working when only one configuration fits the machine.
func TestSelectFromManifest_HeadlessSingleMatch(t *testing.T) {
	dir := t.TempDir()
	manifest := "configurations:\n" +
		"  - name: only\n    init: Only/init.yaml\n    os: " + runtime.GOOS + "\n" +
		"  - name: other\n    init: Other/init.yaml\n    os: never-matches\n"
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewAppConfig()
	app.NoTUI = true // headless: single match should auto-select

	selected, err := selectFromManifest(app, manifestPath)
	if err != nil {
		t.Fatalf("headless single match errored: %v", err)
	}
	if selected != filepath.Join(dir, "Only", "init.yaml") {
		t.Errorf("selected = %s, want Only/init.yaml", selected)
	}
}

// --no-tui with multiple matches and a declared default picks the default
// without prompting - the headless fallback is deterministic, not arbitrary.
func TestSelectFromManifest_HeadlessDefaultWins(t *testing.T) {
	dir := t.TempDir()
	manifest := "configurations:\n" +
		"  - name: desktop\n    init: Desktop/init.yaml\n    os: " + runtime.GOOS + "\n" +
		"  - name: server\n    init: Server/init.yaml\n    os: " + runtime.GOOS + "\n    default: true\n"
	manifestPath := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewAppConfig()
	app.NoTUI = true

	selected, err := selectFromManifest(app, manifestPath)
	if err != nil {
		t.Fatalf("headless default selection errored: %v", err)
	}
	if selected != filepath.Join(dir, "Server", "init.yaml") {
		t.Errorf("selected = %s, want the default Server/init.yaml", selected)
	}
}
