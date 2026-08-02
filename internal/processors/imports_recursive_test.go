package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Imports never recursed. Every blueprint type carried its own copy of the loop —
// ten of them — and each one decoded an imported file and took its items without
// following the imports that file declared. A shared blueprint that imports a base
// blueprint contributed nothing from the base, silently.
//
// Each copy also carried cycle detection that could not fire, because it tracked
// visited paths within one level of a walk that never went deeper.

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func installedNames(calls []exectest.Call) []string {
	var names []string
	for _, call := range calls {
		if len(call.Args) > 0 {
			names = append(names, call.Args[len(call.Args)-1])
		}
	}
	return names
}

func TestImports_FollowANestedChain(t *testing.T) {
	useTestProvider(t)
	// A imports B, B imports C. All three packages must be installed.
	root := writeFiles(t, map[string]string{
		"a.yaml": "packages:\n  - import: b.yaml\n  - name: from-a\n    action: install\n    package_manager: pacman\n",
		"b.yaml": "packages:\n  - import: c.yaml\n  - name: from-b\n    action: install\n    package_manager: pacman\n",
		"c.yaml": "packages:\n  - name: from-c\n    action: install\n    package_manager: pacman\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init); err != nil {
		t.Fatal(err)
	}

	installed := installedNames(rec.Calls)
	for _, want := range []string{"from-a", "from-b", "from-c"} {
		var found bool
		for _, got := range installed {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s was not installed; installed: %v", want, installed)
		}
	}
}

// An import path resolves relative to the file that declares it, so a chain can
// walk into subdirectories.
func TestImports_ResolveRelativeToTheImportingFile(t *testing.T) {
	useTestProvider(t)
	root := writeFiles(t, map[string]string{
		"top.yaml":                 "packages:\n  - import: common/base.yaml\n",
		"common/base.yaml":         "packages:\n  - import: shared/tools.yaml\n",
		"common/shared/tools.yaml": "packages:\n  - name: deep\n    action: install\n    package_manager: pacman\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "top.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init); err != nil {
		t.Fatal(err)
	}

	installed := installedNames(rec.Calls)
	if len(installed) != 1 || installed[0] != "deep" {
		t.Fatalf("nested relative import did not resolve; installed: %v", installed)
	}
}

// A real cycle has to be reported. It could never be reached before.
func TestImports_CycleIsReported(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"a.yaml": "packages:\n  - import: b.yaml\n",
		"b.yaml": "packages:\n  - import: a.yaml\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	err = ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init)
	if err == nil {
		t.Fatal("a cycle between a.yaml and b.yaml was not reported")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should name the problem, got: %v", err)
	}
}

// The same shared file reached from two branches is not a cycle, and must not be
// reported as one.
func TestImports_DiamondIsNotACycle(t *testing.T) {
	useTestProvider(t)
	root := writeFiles(t, map[string]string{
		"top.yaml":   "packages:\n  - import: left.yaml\n  - import: right.yaml\n",
		"left.yaml":  "packages:\n  - import: base.yaml\n",
		"right.yaml": "packages:\n  - import: base.yaml\n",
		"base.yaml":  "packages:\n  - name: shared\n    action: install\n    package_manager: pacman\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "top.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init); err != nil {
		t.Fatalf("a shared file reached twice was treated as a cycle: %v", err)
	}
}

// A missing import is a mistake in the blueprints and must not pass silently.
func TestImports_MissingFileIsReported(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"a.yaml": "packages:\n  - import: nope.yaml\n",
	})

	defer system.SetExecutor(exectest.New())()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init); err == nil {
		t.Fatal("a missing import file was accepted")
	}
}

// An imported file declaring a schema this build cannot read must be refused,
// wherever it sits in the chain. An import is where a file written for a different
// rwr arrives from somebody else's repository.
func TestImports_SchemaVersionEnforcedThroughTheChain(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"a.yaml": "packages:\n  - import: b.yaml\n",
		"b.yaml": "packages:\n  - import: c.yaml\n",
		"c.yaml": "schema_version: 99\npackages:\n  - name: x\n    action: install\n    package_manager: pacman\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	err = ProcessPackages(data, nil, root, "yaml", newTestOSInfo(), init)
	if err == nil {
		t.Fatalf("version 99 two levels down was accepted; %d command(s) ran", len(rec.Calls))
	}
	if len(rec.Calls) != 0 {
		t.Errorf("refused chain still ran %d command(s)", len(rec.Calls))
	}
}

// Import paths resolve relative to the file that declares them, in every
// processor. Six processors (packages among them) used to resolve top-level
// imports against the tree root instead, so `import: ../shared/common.yaml`
// written in packages/base.yaml meant a different file than the same string in
// files/base.yaml — and the spec has always mandated file-relative.
func TestImports_TopLevelResolvesRelativeToTheBlueprintFile(t *testing.T) {
	useTestProvider(t)
	root := writeFiles(t, map[string]string{
		"packages/base.yaml": "packages:\n  - import: ../shared/common.yaml\n",
		"shared/common.yaml": "packages:\n  - name: from-shared\n    action: install\n    package_manager: pacman\n",
	})

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	init := &types.InitConfig{}
	init.Init.Location = root

	data, err := os.ReadFile(filepath.Join(root, "packages", "base.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ProcessPackages(data, nil, filepath.Join(root, "packages"), "yaml", newTestOSInfo(), init); err != nil {
		t.Fatal(err)
	}

	installed := installedNames(rec.Calls)
	if len(installed) != 1 || installed[0] != "from-shared" {
		t.Errorf("installed = %v, want [from-shared] resolved relative to packages/base.yaml", installed)
	}
}
