package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/types"
)

func machineWith(packages ...string) Machine {
	return Machine{Packages: []scan.PackageResult{{Provider: "pacman", Names: packages}}}
}

func planWith(names ...string) *types.Plan {
	plan := &types.Plan{}
	for _, name := range names {
		plan.Resources = append(plan.Resources, types.Resource{
			Processor: types.BlueprintTypePackages, Name: name,
		})
	}
	return plan
}

func TestCompute_AdditionsAndRemovals(t *testing.T) {
	changes := Compute(machineWith("git", "helix"), planWith("git", "vim"), nil)

	byName := map[string]Change{}
	for _, change := range changes {
		byName[change.Name] = change
	}
	if change, ok := byName["helix"]; !ok || change.Removal || change.Provider != "pacman" {
		t.Fatalf("helix = %+v, want a pacman addition", change)
	}
	if change, ok := byName["vim"]; !ok || !change.Removal {
		t.Fatalf("vim = %+v, want a removal", change)
	}
	if _, ok := byName["git"]; ok {
		t.Fatal("declared and present package reported as drift")
	}
}

// A package the journal shows a run applied is not hand-added, whatever the
// current tree says.
func TestCompute_JournalAppliedIsNotDrift(t *testing.T) {
	record := &state.RecordFile{Path: "r", Record: &state.Record{Entries: []state.Entry{{
		Processor: types.BlueprintTypePackages, OK: true, Outcome: "ok",
		Identity: map[string]string{"provider": "pacman", "name": "helix"},
	}}}}
	changes := Compute(machineWith("helix"), planWith(), []*state.RecordFile{record})
	for _, change := range changes {
		if change.Name == "helix" && !change.Removal {
			t.Fatalf("journal-applied package reported as addition: %+v", change)
		}
	}
}

func TestEmitBlocks_StrictDecodes(t *testing.T) {
	machine := machineWith("helix")
	changes := Compute(machine, planWith(), nil)
	block, err := EmitBlocks(changes, machine, "cue")
	if err != nil {
		t.Fatal(err)
	}
	var d types.PackagesData
	if err := helpers.DecodeBlueprintInto([]byte(block), "json", types.BlueprintTypePackages, 0, &d); err != nil {
		t.Fatalf("emitted block does not strict-decode: %v\n%s", err, block)
	}
}

func fixtureTree(t *testing.T) string {
	t.Helper()
	tree := t.TempDir()
	common := filepath.Join(tree, "..", "Common", "packages")
	if err := os.MkdirAll(common, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(common, "base.yaml"), []byte("packages:\n- name: git\n  action: install\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgDir := filepath.Join(tree, "packages")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	main := "packages:\n- import: ../../Common/packages/base.yaml\n- name: vim\n  action: install\n"
	if err := os.WriteFile(filepath.Join(pkgDir, "pacman.yaml"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	return tree
}

// Destination discovery offers the tree's own file and the Common file it
// imports.
func TestDestinations_IncludeImports(t *testing.T) {
	tree := fixtureTree(t)
	destinations, err := Destinations(tree, types.BlueprintTypePackages)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(destinations, "\n")
	if !strings.Contains(joined, "pacman.yaml") || !strings.Contains(joined, "base.yaml") {
		t.Fatalf("destinations = %v", destinations)
	}
}

// Appending writes the destination file's own format and the result
// strict-decodes and keeps existing entries.
func TestAppendEntries(t *testing.T) {
	tree := fixtureTree(t)
	destination := filepath.Join(tree, "packages", "pacman.yaml")
	if err := AppendEntries(destination, types.BlueprintTypePackages, PackageEntries("pacman", []string{"helix"})); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	var d types.PackagesData
	if err := helpers.DecodeBlueprintInto(data, "yaml", types.BlueprintTypePackages, 0, &d); err != nil {
		t.Fatalf("appended file does not strict-decode: %v\n%s", err, data)
	}
	names := map[string]bool{}
	for _, pkg := range d.Packages {
		names[pkg.Name] = true
		for _, n := range pkg.Names {
			names[n] = true
		}
		if pkg.Import != "" {
			names["<import>"] = true
		}
	}
	for _, want := range []string{"vim", "helix", "<import>"} {
		if !names[want] {
			t.Fatalf("missing %s after append: %+v", want, d.Packages)
		}
	}
}
