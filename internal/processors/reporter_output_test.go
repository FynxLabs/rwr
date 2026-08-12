package processors

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// The safety net for the reporter refactor (add-tui task 2): a headless run's
// log output must stay byte-identical to what All() streamed before events
// existed. Written against the pre-refactor code; LogReporter has to keep it
// green.
func TestAll_HeadlessOutputUnchanged(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("init.yaml", "blueprints:\n  format: yaml\n  location: "+dir+"\n")
	write("packages/base.yaml", "packages:\n  - name: git\n    action: install\n    package_manager: pacman\n")
	write("scripts/hello.yaml", "scripts:\n  - name: hello\n    action: run\n    content: \"echo hi\"\n    exec: bash\n")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	bin := "sh"
	if runtime.GOOS == types.OSWindows {
		bin = "cmd"
	}
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": {
			Name:      "pacman",
			Detection: types.DetectionConfig{Binary: bin, Distributions: []string{runtime.GOOS}},
			Commands:  types.CommandConfig{Install: "-S --noconfirm"},
		},
	})()

	// Bootstrap marker: pretend bootstrapped so All() goes straight to the loop.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	initConfig := &types.InitConfig{}
	initConfig.Init.Location = dir
	initConfig.Init.Format = "yaml"
	initConfig.Variables.Flags.Interactive = false
	initConfig.Variables.UserDefined = map[string]interface{}{}
	osInfo := &types.OSInfo{}
	osInfo.System.OS = runtime.GOOS
	osInfo.Tools.Bash = types.ToolInfo{Exists: true, Bin: "/bin/bash"}
	givePackageManager(osInfo, bin)

	if err := All(initConfig, osInfo, []string{"packages", "scripts"}); err != nil {
		t.Fatalf("All: %v", err)
	}

	out := buf.String()
	// The exact per-processor lines today's loop prints, in order.
	wantInOrder := []string{
		"Processing packages",
		"Processing scripts",
		"RWR Run Complete!",
	}
	last := -1
	for _, want := range wantInOrder {
		idx := strings.Index(out, want)
		if idx < 0 {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
		if idx < last {
			t.Fatalf("output order wrong: %q appears before the previous marker:\n%s", want, out)
		}
		last = idx
	}
}

// A headless (non-interactive) run pushes through a failing processor,
// still runs the rest, and exits nonzero. Before task 5 the first error
// aborted the loop - this test fails on that behavior.
func TestAll_NonInteractiveCollectsErrorsAndContinues(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("init.yaml", "blueprints:\n  format: yaml\n  location: "+dir+"\n")
	// The scripts processor RETURNS errors (it does not use the failure
	// ledger), so an unsupported action exercises the push-through path
	// itself, not the ledger's existing continue-and-fail behavior.
	write("scripts/bad.yaml", "scripts:\n  - name: bad\n    action: explode\n    content: x\n")
	write("files/rc.yaml", "files:\n  - name: rc.txt\n    action: create\n    target: "+filepath.Join(dir, "out", "rc.txt")+"\n    content: hello\n")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{})()

	initConfig := &types.InitConfig{}
	initConfig.Init.Location = dir
	initConfig.Init.Format = "yaml"
	initConfig.Variables.Flags.Interactive = false
	initConfig.Variables.UserDefined = map[string]interface{}{}
	osInfo := &types.OSInfo{}
	osInfo.System.OS = runtime.GOOS
	osInfo.Tools.Bash = types.ToolInfo{Exists: true, Bin: "/bin/bash"}
	givePackageManager(osInfo, "sh")

	err := All(initConfig, osInfo, []string{"scripts", "files"})
	if err == nil {
		t.Fatal("All returned nil despite a failing processor")
	}

	// The later processor still ran: the file landed on disk.
	if _, statErr := os.Stat(filepath.Join(dir, "out", "rc.txt")); statErr != nil {
		t.Fatalf("files processor did not run after the scripts failure: %v", statErr)
	}

	// Interactive mode keeps halt-on-first-error.
	initConfig.Variables.Flags.Interactive = true
	if err := All(initConfig, osInfo, []string{"scripts"}); err == nil {
		t.Fatal("interactive run did not halt on the failing processor")
	}
}
