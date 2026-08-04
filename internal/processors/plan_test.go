package processors

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Stage 2 fills provider states and enumerates planned resources per
// processor - the lane counts a progress display needs, computed after
// bootstrap could have installed the manager they depend on.
func TestResolveStage2_ProvidersAndResources(t *testing.T) {
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
	write("packages/base.yaml", "packages:\n  - names: [git, vim]\n    action: install\n    package_manager: pacman\n")
	write("services/ssh.yaml", "services:\n  - name: sshd\n    action: enable\n")

	bin := "sh"
	if runtime.GOOS == types.OSWindows {
		bin = "cmd"
	}
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": {
			Name:      "pacman",
			Elevated:  true,
			Detection: types.DetectionConfig{Binary: bin, Distributions: []string{runtime.GOOS}},
			Commands:  types.CommandConfig{Install: "-S"},
		},
	})()

	initConfig := &types.InitConfig{}
	initConfig.Init.Location = dir
	initConfig.Init.Format = "yaml"
	initConfig.Variables.UserDefined = map[string]interface{}{}

	plan, err := ResolveStage1(initConfig)
	if err != nil {
		t.Fatalf("ResolveStage1: %v", err)
	}
	ResolveStage2(plan)

	foundProvider := false
	for _, p := range plan.Providers {
		if p.Name == "pacman" && p.Available && p.Elevated {
			foundProvider = true
		}
	}
	if !foundProvider {
		t.Errorf("pacman not in provider states: %+v", plan.Providers)
	}

	want := map[string]bool{"packages/git": true, "packages/vim": true, "services/sshd": true}
	for _, r := range plan.Resources {
		delete(want, r.Processor+"/"+r.Name)
		if r.Status != types.StatusPlanned {
			t.Errorf("resource %s status = %s, want planned", r.Name, r.Status)
		}
		if r.Processor == "packages" && r.Provider != "pacman" {
			t.Errorf("package %s provider = %q, want pacman", r.Name, r.Provider)
		}
	}
	if len(want) != 0 {
		t.Errorf("resources missing: %v (got %+v)", want, plan.Resources)
	}
}
