package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Manifests are untrusted input: an init path escaping the repo root is about
// to be resolved and executed, so it is refused at load.
func TestLoadManifest_RefusesEscapingInitPaths(t *testing.T) {
	for name, entry := range map[string]string{
		"dotdot":   "../outside/init.yaml",
		"absolute": "/etc/init.yaml",
	} {
		dir := writeManifest(t, "configurations:\n  - name: bad\n    init: "+entry+"\n")
		_, err := LoadManifest(filepath.Join(dir, "manifest.yaml"))
		if err == nil || !strings.Contains(err.Error(), "outside the repository") {
			t.Errorf("%s: err = %v, want an escape refusal", name, err)
		}
	}

	dir := writeManifest(t, "configurations:\n  - name: ok\n    init: Arch/init.yaml\n")
	if _, err := LoadManifest(filepath.Join(dir, "manifest.yaml")); err != nil {
		t.Errorf("contained path refused: %v", err)
	}
}

// Matchers filter against the detected system: zero, one, and many matches.
func TestMatchManifest(t *testing.T) {
	manifest := &types.Manifest{Configurations: []types.ManifestEntry{
		{Name: "arch-desktop", Init: "Arch/init.yaml", OS: "linux", Family: "arch"},
		{Name: "mac", Init: "macOS/init.yaml", OS: "darwin"},
		{Name: "any-linux", Init: "Common/init.yaml", OS: "linux"},
	}}

	sys := func(osName, distro, arch string) *types.OSInfo {
		info := &types.OSInfo{}
		info.System.OS = osName
		info.System.OSFamily = distro
		info.System.OSArch = arch
		return info
	}

	if got := MatchManifest(manifest, sys("linux", "manjaro", "amd64")); len(got) != 2 {
		t.Errorf("manjaro matches = %d (%v), want 2 (arch-desktop via family + any-linux)", len(got), got)
	}
	if got := MatchManifest(manifest, sys("darwin", "", "arm64")); len(got) != 1 || got[0].Name != "mac" {
		t.Errorf("darwin matches = %v, want [mac]", got)
	}
	if got := MatchManifest(manifest, sys("windows", "", "amd64")); len(got) != 0 {
		t.Errorf("windows matches = %v, want none", got)
	}
}

// A directory with no init file but a manifest resolves to the manifest;
// plain trees keep resolving to their init file (the manifest path only
// activates when no init file exists).
func TestResolveInitSource_ManifestRepoRoot(t *testing.T) {
	dir := writeManifest(t, "configurations:\n  - name: a\n    init: A/init.yaml\n")
	resolved, err := ResolveInitSource(dir)
	if err != nil {
		t.Fatalf("ResolveInitSource: %v", err)
	}
	if filepath.Base(resolved) != "manifest.yaml" {
		t.Errorf("resolved = %s, want the manifest", resolved)
	}

	// An init file wins over the manifest.
	if err := os.WriteFile(filepath.Join(dir, "init.yaml"), []byte("blueprints:\n  format: yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err = ResolveInitSource(dir)
	if err != nil || filepath.Base(resolved) != "init.yaml" {
		t.Errorf("resolved = %s (%v), want init.yaml winning", resolved, err)
	}
}
