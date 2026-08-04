package capture

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/types"
)

func testFindings(t *testing.T) Findings {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("alias ll='ls -l'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return Findings{
		Packages: []scan.PackageResult{
			{Provider: "pacman", Names: []string{"git", "vim"}},
			{Provider: "unfilteredpm", Names: []string{"everything"}, Unfiltered: true},
		},
		Configs: []scan.ConfigResult{
			{Path: filepath.Join(home, ".bashrc"), Rel: ".bashrc", Known: true},
			{Path: filepath.Join(home, ".ssh"), Rel: ".ssh", Known: false},
		},
		Services: []string{"tailscaled"},
		Home:     home,
	}
}

// Defaults: explicit packages and known dotfiles in; unfiltered providers,
// services, and secret-shaped paths out.
func TestDefaults(t *testing.T) {
	selection := Defaults(testFindings(t))
	if len(selection.Packages["pacman"]) != 2 {
		t.Fatalf("pacman = %v", selection.Packages["pacman"])
	}
	if _, ok := selection.Packages["unfilteredpm"]; ok {
		t.Fatal("unfiltered provider pre-selected")
	}
	if len(selection.Configs) != 1 || selection.Configs[0].Rel != ".bashrc" {
		t.Fatalf("configs = %+v", selection.Configs)
	}
	if len(selection.Services) != 0 {
		t.Fatal("services pre-selected")
	}
}

// A capture generates a tree that validates, per format.
func TestGenerate_ValidTree(t *testing.T) {
	findings := testFindings(t)
	selection := Defaults(findings)
	for _, format := range []string{"cue", "yaml"} {
		dir := filepath.Join(t.TempDir(), "out")
		var out bytes.Buffer
		if err := Generate(&out, dir, format, selection, findings, true, &types.OSInfo{}); err != nil {
			t.Fatalf("%s: %v\n%s", format, err, out.String())
		}
		ext := "." + format
		for _, want := range []string{"init" + ext, filepath.Join("packages", "packages"+ext), filepath.Join("files", "files"+ext), filepath.Join("files", "src", ".bashrc"), "manifest" + ext} {
			if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
				t.Fatalf("%s: missing %s", format, want)
			}
		}
		if !strings.Contains(out.String(), "validated") {
			t.Fatalf("%s: no validation confirmation:\n%s", format, out.String())
		}
	}
}

// A selection that emits an invalid tree fails loudly.
func TestGenerate_InvalidTreeFails(t *testing.T) {
	findings := testFindings(t)
	selection := Defaults(findings)
	// A package name that strict validation rejects (leading dash reads as an
	// option to the package manager).
	selection.Packages = map[string][]string{"pacman": {"-bad"}}
	dir := filepath.Join(t.TempDir(), "out")
	var out bytes.Buffer
	err := Generate(&out, dir, "yaml", selection, findings, false, &types.OSInfo{})
	if err == nil {
		t.Fatalf("invalid tree accepted:\n%s", out.String())
	}
}
