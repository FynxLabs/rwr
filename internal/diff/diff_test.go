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
	applies := []state.Entry{{
		Processor: types.BlueprintTypePackages, OK: true, Outcome: "ok",
		Identity: map[string]string{"provider": "pacman", "name": "helix"},
	}}
	changes := Compute(machineWith("helix"), planWith(), applies)
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

// A file rwr itself applied is not drift, and the journal is the only place
// that says so.
//
// A files entry's blueprint name ("nvim-config") has nothing to do with the
// path it lands on, so matching a scanned config against Identity["name"]
// never hit: every dotfile rwr had applied came back as "on the machine, not
// in the tree". The journal records where the thing actually went, which is
// exactly what the scan found.
func TestCompute_JournalPathsAccountForConfigsAndCheckouts(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home:    "/home/me",
		Configs: []scan.ConfigResult{{Path: "/home/me/.config/nvim/init.lua", Rel: ".config/nvim/init.lua", Known: true}},
		Git:     []scan.GitCheckout{{Path: "/home/me/src/rwr", URL: "git@github.com:FynxLabs/rwr.git"}},
	}
	applies := []state.Entry{
		{Processor: types.BlueprintTypeFiles,
			Identity: map[string]string{"name": "nvim-config", "dest": "/home/me/.config/nvim/init.lua"}},
		{Processor: types.BlueprintTypeGit,
			Identity: map[string]string{"name": "rwr-checkout", "target": "/home/me/src/rwr"}},
	}

	if changes := Compute(machine, &types.Plan{}, applies); len(changes) != 0 {
		t.Fatalf("rwr's own applies reported as drift: %+v", changes)
	}
}

// The path comparison is by cleaned path, so a recorded trailing slash or a
// "." segment still matches what the scan found.
func TestCompute_JournalPathMatchIsPathAware(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home: "/home/me",
		Git:  []scan.GitCheckout{{Path: "/home/me/src/rwr", URL: "u"}},
	}
	applies := []state.Entry{
		{Processor: types.BlueprintTypeGit, Identity: map[string]string{"target": "/home/me/src/./rwr/"}},
	}

	if changes := Compute(machine, &types.Plan{}, applies); len(changes) != 0 {
		t.Fatalf("an unclean recorded path failed to match: %+v", changes)
	}
}

// A recorded path only accounts for its own category. A checkout at a path
// must not silence a config at that same path, or vice versa.
func TestCompute_JournalPathsAreScopedToTheirCategory(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home:    "/home/me",
		Configs: []scan.ConfigResult{{Path: "/home/me/thing", Rel: "thing"}},
	}
	// The journal accounts for a *checkout* at that path, not a file.
	applies := []state.Entry{
		{Processor: types.BlueprintTypeGit, Identity: map[string]string{"target": "/home/me/thing"}},
	}

	changes := Compute(machine, &types.Plan{}, applies)
	if len(changes) != 1 || changes[0].Category != types.BlueprintTypeFiles {
		t.Fatalf("a git target silenced a config at the same path: %+v", changes)
	}
}

// Everything the list reports as an addition has to be emittable, or
// `rwr diff --emit` quietly hands back less than it just showed. Configs were
// listed and then dropped - the category the command is most often used for.
func TestEmitBlocks_IncludesEveryListedCategory(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home:     "/home/me",
		Packages: []scan.PackageResult{{Provider: "pacman", Names: []string{"ripgrep"}}},
		Services: []string{"sshd"},
		Git:      []scan.GitCheckout{{Path: "/home/me/src/rwr", URL: "git@github.com:FynxLabs/rwr.git"}},
		Configs:  []scan.ConfigResult{{Path: "/home/me/.config/nvim/init.lua", Rel: ".config/nvim/init.lua", Known: true}},
	}

	changes := Compute(machine, &types.Plan{}, nil)
	out, err := EmitBlocks(changes, machine, "yaml")
	if err != nil {
		t.Fatalf("EmitBlocks: %v", err)
	}

	// Every category the list reported must appear in the emitted blocks.
	seen := map[string]bool{}
	for _, change := range changes {
		seen[change.Category] = true
	}
	for category := range seen {
		if !strings.Contains(out, category+":") {
			t.Errorf("category %q was listed as drift but is missing from --emit output:\n%s", category, out)
		}
	}
}

// Two configs can share a filename - half a dozen tools each own a "config" -
// so the emitted block has to be found by path, not by name.
func TestEmitBlocks_ConfigsSharingAFilenameBothEmit(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home: "/home/me",
		Configs: []scan.ConfigResult{
			{Path: "/home/me/.config/nvim/init.lua", Rel: ".config/nvim/init.lua", Known: true},
			{Path: "/home/me/.config/wezterm/init.lua", Rel: ".config/wezterm/init.lua", Known: true},
		},
	}

	changes := Compute(machine, &types.Plan{}, nil)
	if len(changes) != 2 {
		t.Fatalf("expected both configs as drift, got %+v", changes)
	}

	out, err := EmitBlocks(changes, machine, "yaml")
	if err != nil {
		t.Fatalf("EmitBlocks: %v", err)
	}
	for _, want := range []string{"nvim", "wezterm"} {
		if !strings.Contains(out, want) {
			t.Errorf("emitted output lost the %s config:\n%s", want, out)
		}
	}
}

// A journal entry accounts for one path, not for every file that happens to
// share its basename.
//
// Naming a files entry after the file it carries is the convention, so a
// journaled "init.lua" would otherwise suppress every other init.lua on the
// machine - and half a dozen tools own one. Checking the recorded name before
// the recorded path left exactly the basename collision this change exists to
// remove.
func TestCompute_AJournaledNameDoesNotHideAnotherPath(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home: "/home/me",
		Configs: []scan.ConfigResult{
			{Path: "/home/me/.config/nvim/init.lua", Rel: ".config/nvim/init.lua"},
			{Path: "/home/me/.config/wezterm/init.lua", Rel: ".config/wezterm/init.lua"},
		},
	}
	// The recorded entry is named after its file and covers only the nvim one.
	applies := []state.Entry{
		{Processor: types.BlueprintTypeFiles,
			Identity: map[string]string{"name": "init.lua", "dest": "/home/me/.config/nvim/init.lua"}},
	}

	changes := Compute(machine, &types.Plan{}, applies)
	if len(changes) != 1 {
		t.Fatalf("want only the unjournaled config as drift, got %d: %+v", len(changes), changes)
	}
	if changes[0].Path != "/home/me/.config/wezterm/init.lua" {
		t.Errorf("reported the wrong config: %+v", changes[0])
	}

	// And it still reaches the paste-ready output.
	out, err := EmitBlocks(changes, machine, "yaml")
	if err != nil {
		t.Fatalf("EmitBlocks: %v", err)
	}
	if !strings.Contains(out, "wezterm") {
		t.Errorf("the unjournaled config is missing from --emit output:\n%s", out)
	}
	if strings.Contains(out, "nvim") {
		t.Errorf("the journaled config was emitted as drift:\n%s", out)
	}
}

// The same rule for checkouts: two repos can be cloned into directories with
// the same base name.
func TestCompute_AJournaledCheckoutNameDoesNotHideAnotherPath(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Home: "/home/me",
		Git: []scan.GitCheckout{
			{Path: "/home/me/work/rwr", URL: "git@github.com:FynxLabs/rwr.git"},
			{Path: "/home/me/fork/rwr", URL: "git@github.com:someone/rwr.git"},
		},
	}
	applies := []state.Entry{
		{Processor: types.BlueprintTypeGit,
			Identity: map[string]string{"name": "rwr", "target": "/home/me/work/rwr"}},
	}

	changes := Compute(machine, &types.Plan{}, applies)
	if len(changes) != 1 {
		t.Fatalf("want only the unjournaled checkout as drift, got %d: %+v", len(changes), changes)
	}
	if changes[0].Path != "/home/me/fork/rwr" {
		t.Errorf("reported the wrong checkout: %+v", changes[0])
	}
}

// Packages and services have no location, so a name is what identifies them
// and the journal match has to keep working on it.
func TestCompute_NameMatchingStillAppliesWithoutALocation(t *testing.T) {
	t.Parallel()

	machine := Machine{
		Packages: []scan.PackageResult{{Provider: "pacman", Names: []string{"ripgrep", "fd"}}},
		Services: []string{"sshd", "docker"},
	}
	applies := []state.Entry{
		{Processor: types.BlueprintTypePackages, Identity: map[string]string{"name": "ripgrep"}},
		{Processor: types.BlueprintTypeServices, Identity: map[string]string{"name": "sshd"}},
	}

	changes := Compute(machine, &types.Plan{}, applies)
	if len(changes) != 2 {
		t.Fatalf("want fd and docker as drift, got %d: %+v", len(changes), changes)
	}
	for _, change := range changes {
		if change.Name == "ripgrep" || change.Name == "sshd" {
			t.Errorf("a journaled name-identified resource was reported as drift: %+v", change)
		}
	}
}
