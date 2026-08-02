package processors

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// download/write/copy steps run with the provider's privileges and their dest
// is a template rendered against blueprint values. append/remove_line/
// remove_section/remove have been contained since #163; these are the steps
// that create files, closed the same way.
func TestRepositoryWriteStepsAreContained(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	victim := filepath.Join(t.TempDir(), "cron.d", "evil")

	newProvider := func(step types.ActionStep) *types.Provider {
		return &types.Provider{
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Paths: types.RepositoryPaths{Sources: sourcesDir, Keys: keysDir},
				Add:   types.RepositoryAction{Steps: []types.ActionStep{step}},
			},
		}
	}

	for _, tt := range []struct {
		name string
		step types.ActionStep
	}{
		{name: "write outside", step: types.ActionStep{Action: "write", Dest: victim, Content: "* * * * * root sh"}},
		{name: "download outside", step: types.ActionStep{Action: "download", Source: "https://example.com/x", Dest: victim}},
		{name: "copy outside", step: types.ActionStep{Action: "copy", Source: filepath.Join(sourcesDir, "x"), Dest: victim}},
		{name: "relative dest", step: types.ActionStep{Action: "write", Dest: "sources.list.d/x.list", Content: "c"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer system.SetExecutor(exectest.New())()
			defer system.SetProvidersForTest(map[string]*types.Provider{"fake": newProvider(tt.step)})()

			err := processRepository(types.Repository{
				Name:           "example",
				PackageManager: "fake",
				Action:         "add",
				URL:            "https://example.com/repo",
			}, &types.OSInfo{}, &types.InitConfig{})

			if err == nil || !strings.Contains(err.Error(), "refusing to write") {
				t.Fatalf("err = %v, want a containment refusal", err)
			}
			if _, statErr := os.Stat(victim); statErr == nil {
				t.Fatalf("%s written despite the refusal", victim)
			}
		})
	}

	// A dest inside the declared paths still works.
	t.Run("write inside is allowed", func(t *testing.T) {
		inside := filepath.Join(sourcesDir, "example.list")
		defer system.SetExecutor(exectest.New())()
		defer system.SetProvidersForTest(map[string]*types.Provider{
			"fake": newProvider(types.ActionStep{Action: "write", Dest: inside, Content: "deb https://example.com stable main"}),
		})()

		if err := processRepository(types.Repository{
			Name:           "example",
			PackageManager: "fake",
			Action:         "add",
			URL:            "https://example.com/repo",
		}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
			t.Fatalf("processRepository: %v", err)
		}
		if _, err := os.Stat(inside); err != nil {
			t.Fatalf("contained write did not land: %v", err)
		}
	})
}

// repo.Name is joined into KeyPath, TempKeyPath and provider-templated file
// names, all written with the provider's privileges.
func TestProcessRepository_RefusesTraversalName(t *testing.T) {
	defer system.SetExecutor(exectest.New())()

	for _, name := range []string{"../../../etc/cron.d/x", "a/b", `a\b`, "..", " "} {
		err := processRepository(types.Repository{
			Name:           name,
			PackageManager: "apt",
			Action:         "add",
		}, &types.OSInfo{}, &types.InitConfig{})
		if err == nil || !strings.Contains(err.Error(), "name") {
			t.Errorf("name %q: err = %v, want a name refusal", name, err)
		}
	}
}
