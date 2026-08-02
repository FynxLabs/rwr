package processors

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Install steps staged at fixed, world-known /tmp names any local user could
// pre-create or rewrite between the download and the elevated step executing
// it. {{ .TempDir }} renders to a per-run 0700 directory instead.
func TestProcessPackageManagers_InstallStepsStageInPrivateTempDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\necho hi\n"))
	}))
	defer server.Close()

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"fakepm": {
			Name: "fakepm",
			Detection: types.DetectionConfig{
				// Must not exist: an installed binary short-circuits the install.
				Binary:        "rwr-test-no-such-binary",
				Distributions: []string{runtime.GOOS},
			},
			Install: types.InstallConfig{
				Steps: []types.ActionStep{
					{Action: "download", Source: server.URL + "/install.sh", Dest: "{{ .TempDir }}/fakepm-install.sh"},
					{Action: "command", Exec: "bash", Args: []string{"{{ .TempDir }}/fakepm-install.sh"}},
				},
			},
		},
	})()

	err := ProcessPackageManagers([]types.PackageManagerInfo{
		{Name: "fakepm", Action: "install"},
	}, &types.OSInfo{}, &types.InitConfig{})
	if err != nil {
		t.Fatalf("ProcessPackageManagers: %v", err)
	}

	staged := filepath.Join(packageManagerTempDir(), "fakepm-install.sh")
	if got, readErr := os.ReadFile(staged); readErr != nil || !strings.Contains(string(got), "echo hi") {
		t.Fatalf("staged installer = %q (%v), want the downloaded script in the private dir", got, readErr)
	}

	// The private directory is not reachable by other users.
	if info, statErr := os.Stat(packageManagerTempDir()); statErr != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("staging dir mode = %v (%v), want 0700", info.Mode().Perm(), statErr)
	}

	calls := rec.Find("bash")
	if len(calls) != 1 {
		t.Fatalf("recorded %d bash calls, want 1: %v", len(calls), rec.Calls)
	}
	if got := calls[0].Args[0]; got != staged {
		t.Errorf("bash argv[1] = %q, want the rendered staging path %q", got, staged)
	}
	for _, arg := range calls[0].Argv() {
		if strings.Contains(arg, "{{") {
			t.Errorf("unrendered placeholder reached the command: %v", calls[0])
		}
	}
}

// No shipped install or remove step may name a fixed path under /tmp: a
// world-known name in a shared directory is pre-creatable by any local user.
func TestEmbeddedProviders_NoFixedTmpPathsInInstallSteps(t *testing.T) {
	embedded, err := system.LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders: %v", err)
	}

	for name, provider := range embedded {
		for _, steps := range [][]types.ActionStep{provider.Install.Steps, provider.Remove.Steps} {
			for _, step := range steps {
				fields := append([]string{step.Source, step.Dest, step.Exec, step.Content}, step.Args...)
				for _, field := range fields {
					if strings.Contains(field, "/tmp/") {
						t.Errorf("provider %s stages at a fixed /tmp path: %q — use {{ .TempDir }}", name, field)
					}
				}
			}
		}
	}
}
