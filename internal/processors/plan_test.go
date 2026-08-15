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
	ResolveStage2(plan, nil)

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

func TestEnumerateResourcesCarriesLocationIdentity(t *testing.T) {
	dir := t.TempDir()
	files := enumerateResources(types.BlueprintTypeFiles, types.ResolvedFile{
		Format:   "yaml",
		Resolved: []byte("files:\n  - name: init.lua\n    action: create\n    target: " + filepath.ToSlash(dir) + "/\n"),
	}, "")
	if len(files) != 1 {
		t.Fatalf("file resources = %d, want 1", len(files))
	}
	if want := filepath.Join(dir, "init.lua"); files[0].Location != want {
		t.Errorf("file location = %q, want %q", files[0].Location, want)
	}

	checkout := filepath.Join(dir, "checkout")
	git := enumerateResources(types.BlueprintTypeGit, types.ResolvedFile{
		Format:   "yaml",
		Resolved: []byte("git:\n  - name: source\n    action: clone\n    path: " + filepath.ToSlash(checkout) + "\n"),
	}, "")
	if len(git) != 1 {
		t.Fatalf("git resources = %d, want 1", len(git))
	}
	if git[0].Location != checkout {
		t.Errorf("git location = %q, want %q", git[0].Location, checkout)
	}
}
