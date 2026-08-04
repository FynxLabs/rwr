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

	tempDir, tempErr := packageManagerTempDir()
	if tempErr != nil {
		t.Fatalf("packageManagerTempDir: %v", tempErr)
	}
	staged := filepath.Join(tempDir, "fakepm-install.sh")
	if got, readErr := os.ReadFile(staged); readErr != nil || !strings.Contains(string(got), "echo hi") {
		t.Fatalf("staged installer = %q (%v), want the downloaded script in the private dir", got, readErr)
	}

	// The private directory is not reachable by other users.
	if info, statErr := os.Stat(tempDir); statErr != nil || info.Mode().Perm() != 0o700 {
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

// No shipped install or remove step may stage anywhere another local user can
// reach or predict: not a fixed path under /tmp (pre-creatable by anyone), not
// the operator's CWD (`curl -O`, a bare relative filename), because the staged
// file is usually executed or installed by a later elevated step.
func TestEmbeddedProviders_NoSharedOrCWDStagingInInstallSteps(t *testing.T) {
	embedded, err := system.LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders: %v", err)
	}

	// A path argument is contained when it is inside the per-run private
	// directory or explicitly absolute; a URL is not a path.
	contained := func(p string) bool {
		return strings.HasPrefix(p, "{{ .TempDir }}") ||
			strings.HasPrefix(p, "/") ||
			strings.HasPrefix(p, "~") ||
			strings.Contains(p, "://")
	}

	// File-looking arguments: what a download stages and a later step consumes.
	stagedFile := func(arg string) bool {
		for _, ext := range []string{".pkg", ".sh", ".tar.gz", ".tar.xz", ".zip", ".dmg"} {
			if strings.HasSuffix(arg, ext) {
				return true
			}
		}
		return false
	}

	for name, provider := range embedded {
		for _, steps := range [][]types.ActionStep{provider.Install.Steps, provider.Remove.Steps} {
			for _, step := range steps {
				fields := append([]string{step.Source, step.Dest, step.Exec, step.Content}, step.Args...)
				for _, field := range fields {
					if strings.Contains(field, "/tmp/") {
						t.Errorf("provider %s stages at a fixed /tmp path: %q - use {{ .TempDir }}", name, field)
					}
				}

				if step.Action == "download" && !contained(step.Dest) {
					t.Errorf("provider %s downloads to a relative dest: %q - use {{ .TempDir }}", name, step.Dest)
				}

				for i, arg := range step.Args {
					// curl -O / --remote-name writes into the CWD by construction.
					if step.Exec == "curl" && (arg == "-O" || arg == "--remote-name") {
						t.Errorf("provider %s uses `curl %s`, which stages in the CWD - use a download step with a {{ .TempDir }} dest", name, arg)
					}
					// A bare relative file name consumed or produced by a step
					// resolves against whatever directory rwr happens to run in.
					if stagedFile(arg) && !contained(arg) {
						t.Errorf("provider %s references a relative staged file: args[%d] = %q - use {{ .TempDir }}", name, i, arg)
					}
				}
			}
		}
	}
}
