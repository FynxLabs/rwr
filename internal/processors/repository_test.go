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

// providerForTest returns a shipped provider definition with only the parts a test
// cannot use replaced: the binary it detects, the paths it writes to, and its
// elevation. The steps and templates under test are the ones rwr actually ships.
func providerForTest(t *testing.T, name, sourcesDir, keysDir string) *types.Provider {
	t.Helper()

	embedded, err := system.LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders: %v", err)
	}
	provider, ok := embedded[name]
	if !ok {
		t.Fatalf("embedded providers do not include %s", name)
	}

	// "go" stands in for the package manager binary: the test runner is running
	// under it, so detection succeeds on any platform without depending on what the
	// host has installed.
	provider.Detection.Binary = "go"
	provider.Detection.Files = nil
	provider.Detection.Distributions = []string{runtime.GOOS}
	provider.Elevated = false
	provider.Repository.Paths.Sources = sourcesDir
	provider.Repository.Paths.Keys = keysDir

	return provider
}

func aptProviderForTest(t *testing.T, sourcesDir, keysDir string) *types.Provider {
	t.Helper()
	return providerForTest(t, "apt", sourcesDir, keysDir)
}

// repoDirs returns a sources and a keys directory under one parent, so a test can
// assert that a removal stayed inside them.
func repoDirs(t *testing.T) (sourcesDir, keysDir string) {
	t.Helper()

	root := t.TempDir()
	sourcesDir = filepath.Join(root, "sources.list.d")
	keysDir = filepath.Join(root, "keyrings")
	for _, dir := range []string{sourcesDir, keysDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return sourcesDir, keysDir
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// assertNoPlaceholders fails if any recorded argv still carries an unrendered
// template, which is what a missing step field or a missing data key looks like
// once it reaches the system.
func assertNoPlaceholders(t *testing.T, rec *exectest.Recorder) {
	t.Helper()
	for _, call := range rec.Calls {
		for _, arg := range call.Argv() {
			if strings.Contains(arg, "{{") {
				t.Errorf("unrendered placeholder in command %v", call)
			}
		}
	}
}

func TestProcessRepositories_AptAddRendersProviderTemplates(t *testing.T) {
	key := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("key-material")); err != nil {
			t.Errorf("serving key: %v", err)
		}
	}))
	defer key.Close()

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	t.Setenv("TEMP", tempDir)
	t.Setenv("TMP", tempDir)

	sourcesDir := filepath.Join(t.TempDir(), "sources.list.d")
	keysDir := filepath.Join(t.TempDir(), "keyrings")
	for _, dir := range []string{sourcesDir, keysDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"apt": aptProviderForTest(t, sourcesDir, keysDir),
	})()

	repo := types.Repository{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "add",
		URL:            "https://download.docker.com/linux/ubuntu",
		KeyURL:         key.URL + "/gpg",
		Arch:           "amd64",
		Channel:        "jammy",
		Component:      "stable",
	}

	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	keyPath := filepath.Join(keysDir, "docker.gpg")
	// The staging path is inside the run's private 0700 directory, not a
	// world-known name under /tmp any local user could pre-create or swap
	// between the download and the dearmor that imports it as root.
	stagingDir, err := repositoryTempDir()
	if err != nil {
		t.Fatalf("repositoryTempDir: %v", err)
	}
	tempKeyPath := filepath.Join(stagingDir, "docker.gpg")

	// The key is downloaded to the temporary path the dearmor step reads from.
	if _, err := os.Stat(tempKeyPath); err != nil {
		t.Fatalf("downloaded key: %v", err)
	}

	gpgCalls := rec.Find("gpg")
	if len(gpgCalls) != 1 {
		t.Fatalf("recorded %d gpg calls, want 1: %v", len(gpgCalls), rec.Calls)
	}
	wantArgv := []string{"gpg", "--yes", "--dearmor", "-o", keyPath, tempKeyPath}
	if got := gpgCalls[0].Argv(); !equalStrings(got, wantArgv) {
		t.Errorf("gpg argv = %#v, want %#v", got, wantArgv)
	}

	content, err := os.ReadFile(filepath.Join(sourcesDir, "docker.list")) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered sources file: %v", err)
	}
	wantContent := "deb [arch=amd64 signed-by=" + keyPath + "] https://download.docker.com/linux/ubuntu jammy stable"
	if string(content) != wantContent {
		t.Errorf("sources file = %q, want %q", content, wantContent)
	}

	// Nothing anywhere in the run may still carry an unrendered placeholder.
	for _, call := range rec.Calls {
		for _, arg := range call.Argv() {
			if strings.Contains(arg, "{{") {
				t.Errorf("unrendered placeholder in command %v", call)
			}
		}
	}
	if strings.Contains(string(content), "{{") {
		t.Errorf("unrendered placeholder in sources file %q", content)
	}
}

// The trailing repository refresh execs argv, so the update command may not
// contain shell operators: apt would take "&&" and "upgrade" as package names.
func TestProcessRepositories_UpdateCommandIsPlainArgv(t *testing.T) {
	sourcesDir := t.TempDir()
	keysDir := t.TempDir()

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	provider := aptProviderForTest(t, sourcesDir, keysDir)
	defer system.SetProvidersForTest(map[string]*types.Provider{"apt": provider})()

	if err := processRepositories(nil, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}
	if got := rec.Calls[0].Args; !equalStrings(got, []string{"update"}) {
		t.Errorf("update args = %#v, want [update]", got)
	}
}

// A placeholder rwr has no value for is a provider or blueprint defect. Rendering
// it as "<no value>" — or passing it through verbatim, as rwr used to — turns it
// into a file path or a URL instead.
func TestProcessRepositories_UnknownPlaceholderIsAnError(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	defer system.SetProvidersForTest(map[string]*types.Provider{
		"fake": {
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Add: types.RepositoryAction{
					Steps: []types.ActionStep{{
						Action: "command",
						Exec:   "fake",
						Args:   []string{"add", "{{ .NotAThing }}"},
					}},
				},
			},
		},
	})()

	err := processRepository(types.Repository{
		Name:           "example",
		PackageManager: "fake",
		Action:         "add",
		URL:            "https://example.com/repo",
	}, &types.OSInfo{}, &types.InitConfig{})

	if err == nil {
		t.Fatal("processRepositories succeeded, want an error for the unknown placeholder")
	}
	if !strings.Contains(err.Error(), "NotAThing") {
		t.Errorf("error = %v, want it to name the unknown placeholder", err)
	}
	if len(rec.Calls) != 0 {
		t.Errorf("ran %v despite the unrenderable step", rec.Calls)
	}
}

// Every shipped provider spells a removal as `action = "remove"` with a `path`,
// so a repository blueprint asking for a removal used to fail outright.
func TestProcessRepositories_AptRemoveDeletesRenderedPaths(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	sourcesFile := filepath.Join(sourcesDir, "docker.list")
	keyFile := filepath.Join(keysDir, "docker.gpg")
	writeTestFile(t, sourcesFile)
	writeTestFile(t, keyFile)

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"apt": aptProviderForTest(t, sourcesDir, keysDir),
	})()

	repo := types.Repository{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "remove",
		URL:            "https://download.docker.com/linux/ubuntu",
	}

	// Twice: `rwr all` is run repeatedly, and the second removal has nothing left
	// to delete.
	for run := 1; run <= 2; run++ {
		if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
			t.Fatalf("processRepositories (run %d): %v", run, err)
		}
		for _, path := range []string{sourcesFile, keyFile} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("run %d: %s still present (stat err %v)", run, path, err)
			}
		}
	}

	assertNoPlaceholders(t, rec)
}

// A provider definition is a template rendered against blueprint values, and the
// delete runs as root. Neither may aim it at a file the provider does not own.
func TestProcessRepositories_RemoveRefusesPathOutsideProviderPaths(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	victim := filepath.Join(t.TempDir(), "passwd")
	writeTestFile(t, victim)

	provider := aptProviderForTest(t, sourcesDir, keysDir)
	provider.Repository.Remove.Steps = []types.ActionStep{{Action: "remove", Path: victim}}

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"apt": provider})()

	err := processRepository(types.Repository{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "remove",
	}, &types.OSInfo{}, &types.InitConfig{})

	if err == nil {
		t.Fatal("processRepositories succeeded, want a refusal for a path outside the provider's paths")
	}
	if !strings.Contains(err.Error(), "outside the provider's repository paths") {
		t.Errorf("error = %v, want it to name the containment refusal", err)
	}
	if _, statErr := os.Stat(victim); statErr != nil {
		t.Errorf("removed %s despite the refusal: %v", victim, statErr)
	}
}

func TestProcessRepositories_RemoveRefusesRelativePath(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	provider := aptProviderForTest(t, sourcesDir, keysDir)
	provider.Repository.Remove.Steps = []types.ActionStep{{Action: "remove", Path: "sources.list.d/{{ .Name }}.list"}}

	defer system.SetExecutor(exectest.New())()
	defer system.SetProvidersForTest(map[string]*types.Provider{"apt": provider})()

	err := processRepository(types.Repository{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "remove",
	}, &types.OSInfo{}, &types.InitConfig{})

	if err == nil || !strings.Contains(err.Error(), "not absolute") {
		t.Fatalf("error = %v, want a refusal naming the relative path", err)
	}
}

func TestProcessRepositories_RemoveDryRunKeepsFiles(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	sourcesFile := filepath.Join(sourcesDir, "docker.list")
	writeTestFile(t, sourcesFile)

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	defer system.SetExecutor(exectest.New())()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"apt": aptProviderForTest(t, sourcesDir, keysDir),
	})()

	if err := processRepositories([]types.Repository{{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "remove",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	if _, err := os.Stat(sourcesFile); err != nil {
		t.Errorf("dry run removed %s: %v", sourcesFile, err)
	}
}

// Sources lists and keyrings are root-owned, so an elevated provider's removal has
// to go out through the same elevation its writes used rather than an in-process
// unlink that would fail with EACCES.
func TestProcessRepositories_ElevatedRemoveRunsElevatedRm(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	sourcesFile := filepath.Join(sourcesDir, "docker.list")
	writeTestFile(t, sourcesFile)

	provider := aptProviderForTest(t, sourcesDir, keysDir)
	provider.Elevated = true

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"apt": provider})()

	if err := processRepositories([]types.Repository{{
		Name:           "docker",
		PackageManager: "apt",
		Action:         "remove",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	rmCalls := rec.Find("rm")
	if len(rmCalls) != 2 {
		t.Fatalf("recorded %d rm calls, want 2: %v", len(rmCalls), rec.Calls)
	}
	wantArgv := []string{"rm", "-f", "--", sourcesFile}
	if got := rmCalls[0].Argv(); !equalStrings(got, wantArgv) {
		t.Errorf("rm argv = %#v, want %#v", got, wantArgv)
	}
	if !rmCalls[0].Elevated {
		t.Error("rm was not elevated")
	}
	assertNoPlaceholders(t, rec)
}

// pacmanProviderForTest also redirects pacman.conf, which the shipped
// definition now derives from repository.paths rather than hardcoding.
func pacmanProviderForTest(t *testing.T, sourcesDir, keysDir, confPath string) *types.Provider {
	t.Helper()

	provider := providerForTest(t, "pacman", sourcesDir, keysDir)
	provider.Repository.Paths.Config = confPath
	return provider
}

func TestProcessRepositories_PacmanAddRendersKeyID(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confPath := filepath.Join(t.TempDir(), "pacman.conf")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": pacmanProviderForTest(t, sourcesDir, keysDir, confPath),
	})()

	repo := types.Repository{
		Name:           "chaotic-aur",
		PackageManager: "pacman",
		Action:         "add",
		URL:            "https://geo-mirror.chaotic.cx/$repo/$arch",
		KeyID:          "3056513887B78AEB",
	}

	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	content, err := os.ReadFile(confPath) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered pacman.conf: %v", err)
	}
	wantContent := "[chaotic-aur]\nServer = https://geo-mirror.chaotic.cx/$repo/$arch\n"
	if string(content) != wantContent {
		t.Errorf("pacman.conf = %q, want %q", content, wantContent)
	}

	keyCalls := rec.Find("pacman-key")
	if len(keyCalls) != 2 {
		t.Fatalf("recorded %d pacman-key calls, want 2: %v", len(keyCalls), rec.Calls)
	}
	wantArgv := [][]string{
		{"pacman-key", "--recv-keys", "3056513887B78AEB"},
		{"pacman-key", "--lsign-key", "3056513887B78AEB"},
	}
	for i, want := range wantArgv {
		if got := keyCalls[i].Argv(); !equalStrings(got, want) {
			t.Errorf("pacman-key argv = %#v, want %#v", got, want)
		}
	}

	assertNoPlaceholders(t, rec)
	if strings.Contains(string(content), "{{") {
		t.Errorf("unrendered placeholder in pacman.conf %q", content)
	}
}

// Adding a pacman repository used to write pacman.conf, which replaced it: every
// other repository and every option the machine had were destroyed by a single
// `rwr all`. The section is appended instead, and appending it again on the next
// run must not duplicate it.
func TestProcessRepositories_PacmanAddPreservesExistingConf(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confPath := filepath.Join(t.TempDir(), "pacman.conf")

	existing := "[options]\nHoldPkg = pacman glibc\nParallelDownloads = 5\n\n[core]\nInclude = /etc/pacman.d/mirrorlist\n"
	if err := os.WriteFile(confPath, []byte(existing), 0o600); err != nil {
		t.Fatalf("seeding pacman.conf: %v", err)
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": pacmanProviderForTest(t, sourcesDir, keysDir, confPath),
	})()

	repo := types.Repository{
		Name:           "chaotic-aur",
		PackageManager: "pacman",
		Action:         "add",
		URL:            "https://geo-mirror.chaotic.cx/$repo/$arch",
		KeyID:          "3056513887B78AEB",
	}

	section := "[chaotic-aur]\nServer = https://geo-mirror.chaotic.cx/$repo/$arch\n"
	for run := 1; run <= 2; run++ {
		if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
			t.Fatalf("processRepositories (run %d): %v", run, err)
		}

		content, err := os.ReadFile(confPath) // #nosec G304 -- test-owned temp dir
		if err != nil {
			t.Fatalf("run %d: reading pacman.conf: %v", run, err)
		}
		if !strings.HasPrefix(string(content), existing) {
			t.Fatalf("run %d: pacman.conf lost its existing content: %q", run, content)
		}
		if got := strings.Count(string(content), "[chaotic-aur]"); got != 1 {
			t.Errorf("run %d: pacman.conf holds %d chaotic-aur sections, want 1: %q", run, got, content)
		}
		if want := existing + section; string(content) != want {
			t.Errorf("run %d: pacman.conf = %q, want %q", run, content, want)
		}
	}

	assertNoPlaceholders(t, rec)
}

// Removing the repository takes its section back out of pacman.conf and leaves
// everything else in place, on the first run and on every one after it.
func TestProcessRepositories_PacmanRemoveDropsOnlyItsSection(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confPath := filepath.Join(t.TempDir(), "pacman.conf")

	existing := "[options]\nParallelDownloads = 5\n\n[core]\nInclude = /etc/pacman.d/mirrorlist\n"
	if err := os.WriteFile(confPath, []byte(existing+"[chaotic-aur]\nServer = https://geo-mirror.chaotic.cx/$repo/$arch\n"), 0o600); err != nil {
		t.Fatalf("seeding pacman.conf: %v", err)
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": pacmanProviderForTest(t, sourcesDir, keysDir, confPath),
	})()

	repo := types.Repository{
		Name:           "chaotic-aur",
		PackageManager: "pacman",
		Action:         "remove",
		KeyID:          "3056513887B78AEB",
	}

	for run := 1; run <= 2; run++ {
		if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
			t.Fatalf("processRepositories (run %d): %v", run, err)
		}

		content, err := os.ReadFile(confPath) // #nosec G304 -- test-owned temp dir
		if err != nil {
			t.Fatalf("run %d: reading pacman.conf: %v", run, err)
		}
		if strings.Contains(string(content), "chaotic-aur") {
			t.Errorf("run %d: pacman.conf still holds the section: %q", run, content)
		}
		if !strings.Contains(string(content), "[core]") || !strings.Contains(string(content), "ParallelDownloads = 5") {
			t.Errorf("run %d: pacman.conf lost unrelated content: %q", run, content)
		}
	}

	keyCalls := rec.Find("pacman-key")
	if len(keyCalls) != 2 {
		t.Fatalf("recorded %d pacman-key calls over two runs, want 2: %v", len(keyCalls), rec.Calls)
	}
	wantArgv := []string{"pacman-key", "--delete", "3056513887B78AEB"}
	if got := keyCalls[0].Argv(); !equalStrings(got, wantArgv) {
		t.Errorf("pacman-key argv = %#v, want %#v", got, wantArgv)
	}
	assertNoPlaceholders(t, rec)
}

// An edit step may only touch a file the provider declares, however the step's
// path template renders.
func TestProcessRepositories_AppendRefusesPathOutsideProviderPaths(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confPath := filepath.Join(t.TempDir(), "pacman.conf")

	victim := filepath.Join(t.TempDir(), "sudoers")
	writeTestFile(t, victim)

	provider := pacmanProviderForTest(t, sourcesDir, keysDir, confPath)
	for i, step := range provider.Repository.Add.Steps {
		if step.Action == "append" {
			provider.Repository.Add.Steps[i].Path = victim
		}
	}

	defer system.SetExecutor(exectest.New())()
	defer system.SetProvidersForTest(map[string]*types.Provider{"pacman": provider})()

	err := processRepository(types.Repository{
		Name:           "chaotic-aur",
		PackageManager: "pacman",
		Action:         "add",
		URL:            "https://example.com/repo",
	}, &types.OSInfo{}, &types.InitConfig{})

	if err == nil || !strings.Contains(err.Error(), "outside the provider's repository paths") {
		t.Fatalf("error = %v, want a refusal naming the containment check", err)
	}

	content, readErr := os.ReadFile(victim) // #nosec G304 -- test-owned temp dir
	if readErr != nil || string(content) != "existing" {
		t.Errorf("%s = %q (err %v), want it untouched", victim, content, readErr)
	}
}

func TestProcessRepositories_AppendDryRunLeavesFileAlone(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confPath := filepath.Join(t.TempDir(), "pacman.conf")
	if err := os.WriteFile(confPath, []byte("[options]\n"), 0o600); err != nil {
		t.Fatalf("seeding pacman.conf: %v", err)
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	defer system.SetExecutor(exectest.New())()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"pacman": pacmanProviderForTest(t, sourcesDir, keysDir, confPath),
	})()

	if err := processRepositories([]types.Repository{{
		Name:           "chaotic-aur",
		PackageManager: "pacman",
		Action:         "add",
		URL:            "https://example.com/repo",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	content, err := os.ReadFile(confPath) // #nosec G304 -- test-owned temp dir
	if err != nil || string(content) != "[options]\n" {
		t.Errorf("dry run changed pacman.conf to %q (err %v)", content, err)
	}
}

// chocolatey adds the source once unconditionally and once more with
// credentials. With the condition dropped both ran, and the second added the
// same source again with empty --user= and --password=.
func TestProcessRepositories_ChocolateyAddWithoutCredentialsRunsOnce(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"chocolatey": providerForTest(t, "chocolatey", sourcesDir, keysDir),
	})()

	if err := processRepositories([]types.Repository{{
		Name:           "internal",
		PackageManager: "chocolatey",
		Action:         "add",
		URL:            "https://nuget.example.com/api/v2",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	calls := rec.Find("choco")
	if len(calls) != 1 {
		t.Fatalf("recorded %d choco calls, want 1: %v", len(calls), rec.Calls)
	}
	want := []string{"choco", "source", "add", "--name=internal", "--source=https://nuget.example.com/api/v2"}
	if got := calls[0].Argv(); !equalStrings(got, want) {
		t.Errorf("choco argv = %#v, want %#v", got, want)
	}
	assertNoPlaceholders(t, rec)
}

// With credentials the authenticated step is the extra one that runs.
func TestProcessRepositories_ChocolateyAddWithCredentialsRunsAuthenticatedStep(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"chocolatey": providerForTest(t, "chocolatey", sourcesDir, keysDir),
	})()

	if err := processRepositories([]types.Repository{{
		Name:           "internal",
		PackageManager: "chocolatey",
		Action:         "add",
		URL:            "https://nuget.example.com/api/v2",
		Username:       "build",
		Password:       "s3cret",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	calls := rec.Find("choco")
	if len(calls) != 2 {
		t.Fatalf("recorded %d choco calls, want 2: %v", len(calls), rec.Calls)
	}
	want := []string{"choco", "source", "add", "--name=internal", "--source=https://nuget.example.com/api/v2", "--user=build", "--password=s3cret"}
	if got := calls[1].Argv(); !equalStrings(got, want) {
		t.Errorf("choco argv = %#v, want %#v", got, want)
	}
}

// flatpak's remote-add steps are mutually exclusive: --user and --system. Both
// ran when the condition was dropped, so a system remote was added on top of the
// user one on every run.
func TestProcessRepositories_FlatpakAddRunsOneRemoteAdd(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	for _, tt := range []struct {
		name     string
		elevated bool
		wantFlag string
	}{
		{name: "user", elevated: false, wantFlag: "--user"},
		{name: "system", elevated: true, wantFlag: "--system"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := providerForTest(t, "flatpak", sourcesDir, keysDir)
			provider.Elevated = tt.elevated

			rec := exectest.New()
			defer system.SetExecutor(rec)()
			defer system.SetProvidersForTest(map[string]*types.Provider{"flatpak": provider})()

			if err := processRepositories([]types.Repository{{
				Name:           "flathub",
				PackageManager: "flatpak",
				Action:         "add",
				URL:            "https://dl.flathub.org/repo/flathub.flatpakrepo",
			}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
				t.Fatalf("processRepositories: %v", err)
			}

			// The unconditional remote-add plus exactly one of --user/--system.
			calls := rec.Find("flatpak")
			if len(calls) != 2 {
				t.Fatalf("recorded %d flatpak calls, want 2: %v", len(calls), rec.Calls)
			}
			want := []string{"flatpak", "remote-add", "--if-not-exists", tt.wantFlag, "flathub", "https://dl.flathub.org/repo/flathub.flatpakrepo"}
			if got := calls[1].Argv(); !equalStrings(got, want) {
				t.Errorf("flatpak argv = %#v, want %#v", got, want)
			}
		})
	}
}

// A condition rwr cannot derive is a provider defect. Treating it as false is
// how the dropped conditions went unnoticed, so it has to name the predicate and
// stop.
func TestProcessRepositories_UnknownConditionPredicateIsAnError(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	defer system.SetProvidersForTest(map[string]*types.Provider{
		"fake": {
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Add: types.RepositoryAction{
					Steps: []types.ActionStep{{
						Action:    "command",
						Exec:      "fake",
						Args:      []string{"add"},
						Condition: "{{ .HasSomethingUnknowable }}",
					}},
				},
			},
		},
	})()

	err := processRepository(types.Repository{
		Name:           "example",
		PackageManager: "fake",
		Action:         "add",
		URL:            "https://example.com/repo",
	}, &types.OSInfo{}, &types.InitConfig{})

	if err == nil {
		t.Fatal("processRepositories succeeded, want an error for the underivable condition")
	}
	if !strings.Contains(err.Error(), "HasSomethingUnknowable") {
		t.Errorf("error = %v, want it to name the unknown predicate", err)
	}
	if len(rec.Calls) != 0 {
		t.Errorf("ran %v despite the unevaluable condition", rec.Calls)
	}
}

// A false condition is not an error, and the step it gates does not run — even
// when that step references data this repository does not carry.
func TestProcessRepositories_FalseConditionSkipsStep(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	defer system.SetProvidersForTest(map[string]*types.Provider{
		"fake": {
			Name: "fake",
			Detection: types.DetectionConfig{
				Binary:        "go",
				Distributions: []string{runtime.GOOS},
			},
			Repository: types.RepositoryConfig{
				Add: types.RepositoryAction{
					Steps: []types.ActionStep{{
						Action:    "command",
						Exec:      "fake",
						Args:      []string{"import", "{{ .NotAThing }}"},
						Condition: "{{ .HasKey }}",
					}},
				},
			},
		},
	})()

	if err := processRepositories([]types.Repository{{
		Name:           "example",
		PackageManager: "fake",
		Action:         "add",
		URL:            "https://example.com/repo",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	if len(rec.Calls) != 0 {
		t.Errorf("ran %v despite the unmet condition", rec.Calls)
	}
}

// apk keeps its repositories in a single file, so an add appends a line to it
// and a remove takes that line back out without disturbing the others.
func TestProcessRepositories_ApkAddAndRemoveEditRepositoriesFile(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	reposFile := filepath.Join(sourcesDir, "repositories")
	existing := "https://dl-cdn.alpinelinux.org/alpine/v3.19/main\nhttps://dl-cdn.alpinelinux.org/alpine/v3.19/community\n"
	if err := os.WriteFile(reposFile, []byte(existing), 0o600); err != nil {
		t.Fatalf("seeding repositories: %v", err)
	}

	provider := providerForTest(t, "apk", sourcesDir, keysDir)
	provider.Repository.Paths.Sources = reposFile

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"apk": provider})()

	repo := types.Repository{
		Name:           "testing",
		PackageManager: "apk",
		Action:         "add",
		URL:            "https://dl-cdn.alpinelinux.org/alpine/edge/testing",
	}

	for run := 1; run <= 2; run++ {
		if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
			t.Fatalf("processRepositories (add run %d): %v", run, err)
		}
		content, err := os.ReadFile(reposFile) // #nosec G304 -- test-owned temp dir
		if err != nil {
			t.Fatalf("run %d: reading repositories: %v", run, err)
		}
		if want := existing + repo.URL + "\n"; string(content) != want {
			t.Errorf("run %d: repositories = %q, want %q", run, content, want)
		}
	}

	// No key was configured, so the key download must not have run.
	if _, err := os.Stat(filepath.Join(keysDir, "testing.gpg")); !os.IsNotExist(err) {
		t.Errorf("downloaded a key for a repository that configures none (stat err %v)", err)
	}

	repo.Action = "remove"
	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (remove): %v", err)
	}
	content, err := os.ReadFile(reposFile) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading repositories after remove: %v", err)
	}
	if string(content) != existing {
		t.Errorf("repositories = %q, want %q", content, existing)
	}

	assertNoPlaceholders(t, rec)
}

func TestProcessRepositories_DnfAddAndRemove(t *testing.T) {
	key := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("key-material")); err != nil {
			t.Errorf("serving key: %v", err)
		}
	}))
	defer key.Close()

	sourcesDir, keysDir := repoDirs(t)
	keyPath := filepath.Join(keysDir, "docker-ce.gpg")
	repoFile := filepath.Join(sourcesDir, "docker-ce.repo")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"dnf": providerForTest(t, "dnf", sourcesDir, keysDir),
	})()

	repo := types.Repository{
		Name:           "docker-ce",
		PackageManager: "dnf",
		Action:         "add",
		Description:    "Docker CE Stable",
		URL:            "https://download.docker.com/linux/fedora/$releasever/$basearch/stable",
		KeyURL:         key.URL + "/gpg",
		KeyID:          "621E9F35",
	}

	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (add): %v", err)
	}

	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("downloaded key: %v", err)
	}

	rpmCalls := rec.Find("rpm")
	if len(rpmCalls) != 1 {
		t.Fatalf("recorded %d rpm calls, want 1: %v", len(rpmCalls), rec.Calls)
	}
	// The import reads from the run's private staging directory, mirroring
	// apt: the key reaches its final /etc path only through the copy step.
	dnfStagingDir, err := repositoryTempDir()
	if err != nil {
		t.Fatalf("repositoryTempDir: %v", err)
	}
	if got, want := rpmCalls[0].Argv(), []string{"rpm", "--import", filepath.Join(dnfStagingDir, "docker-ce.gpg")}; !equalStrings(got, want) {
		t.Errorf("rpm argv = %#v, want %#v", got, want)
	}

	content, err := os.ReadFile(repoFile) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered repo file: %v", err)
	}
	wantContent := "[docker-ce]\nname=Docker CE Stable\nbaseurl=" + repo.URL +
		"\nenabled=1\ngpgcheck=1\ngpgkey=" + keyPath + "\n"
	if string(content) != wantContent {
		t.Errorf("repo file = %q, want %q", content, wantContent)
	}
	if strings.Contains(string(content), "{{") {
		t.Errorf("unrendered placeholder in repo file %q", content)
	}

	// The same repository removed again: the .repo file goes, and the imported key
	// is erased by ID.
	rec.Calls = nil

	repo.Action = "remove"
	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (remove): %v", err)
	}

	if _, err := os.Stat(repoFile); !os.IsNotExist(err) {
		t.Errorf("%s still present (stat err %v)", repoFile, err)
	}

	rpmCalls = rec.Find("rpm")
	if len(rpmCalls) != 1 {
		t.Fatalf("recorded %d rpm calls on remove, want 1: %v", len(rpmCalls), rec.Calls)
	}
	if got, want := rpmCalls[0].Argv(), []string{"rpm", "--erase", "gpg-pubkey-621E9F35"}; !equalStrings(got, want) {
		t.Errorf("rpm argv = %#v, want %#v", got, want)
	}

	assertNoPlaceholders(t, rec)
}

// xbps and emerge named their repository directory "repos", which decodes into
// nothing: their add step wrote "{{ .SourcesPath }}/{{ .Name }}.conf" to "/name.conf"
// and their remove step then refused to touch it.
func TestProcessRepositories_XbpsAddAndRemove(t *testing.T) {
	key := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("key-material")); err != nil {
			t.Errorf("serving key: %v", err)
		}
	}))
	defer key.Close()

	sourcesDir, keysDir := repoDirs(t)
	confFile := filepath.Join(sourcesDir, "void-nonfree.conf")
	keyFile := filepath.Join(keysDir, "void-nonfree.gpg")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"xbps": providerForTest(t, "xbps", sourcesDir, keysDir),
	})()

	repo := types.Repository{
		Name:           "void-nonfree",
		PackageManager: "xbps",
		Action:         "add",
		URL:            "https://repo-default.voidlinux.org/current/nonfree",
		KeyURL:         key.URL + "/gpg",
	}

	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (add): %v", err)
	}

	content, err := os.ReadFile(confFile) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered conf: %v", err)
	}
	if want := "repository=" + repo.URL + "\n"; string(content) != want {
		t.Errorf("conf = %q, want %q", content, want)
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Fatalf("downloaded key: %v", err)
	}

	repo.Action = "remove"
	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (remove): %v", err)
	}
	for _, path := range []string{confFile, keyFile} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s still present (stat err %v)", path, err)
		}
	}

	assertNoPlaceholders(t, rec)
}

func TestProcessRepositories_EmergeAddAndRemove(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	confFile := filepath.Join(sourcesDir, "guru.conf")

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{
		"emerge": providerForTest(t, "emerge", sourcesDir, keysDir),
	})()

	repo := types.Repository{
		Name:           "guru",
		PackageManager: "emerge",
		Action:         "add",
		URL:            "https://github.com/gentoo-mirror/guru.git",
		OverlayPath:    "/var/db/repos/guru",
		SyncType:       "git",
	}

	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (add): %v", err)
	}

	content, err := os.ReadFile(confFile) // #nosec G304 -- test-owned temp dir
	if err != nil {
		t.Fatalf("reading rendered conf: %v", err)
	}
	wantContent := "[guru]\nlocation = /var/db/repos/guru\nsync-type = git\nsync-uri = " + repo.URL + "\nauto-sync = yes\n"
	if string(content) != wantContent {
		t.Errorf("conf = %q, want %q", content, wantContent)
	}

	emaint := rec.Find("emaint")
	if len(emaint) != 1 {
		t.Fatalf("recorded %d emaint calls, want 1: %v", len(emaint), rec.Calls)
	}
	if got, want := emaint[0].Argv(), []string{"emaint", "sync", "-r", "guru"}; !equalStrings(got, want) {
		t.Errorf("emaint argv = %#v, want %#v", got, want)
	}

	rec.Calls = nil
	repo.Action = "remove"
	if err := processRepositories([]types.Repository{repo}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories (remove): %v", err)
	}
	if _, err := os.Stat(confFile); !os.IsNotExist(err) {
		t.Errorf("%s still present (stat err %v)", confFile, err)
	}

	rm := rec.Find("rm")
	if len(rm) != 1 {
		t.Fatalf("recorded %d rm calls, want 1: %v", len(rm), rec.Calls)
	}
	if got, want := rm[0].Argv(), []string{"rm", "-rf", "/var/db/repos/guru"}; !equalStrings(got, want) {
		t.Errorf("rm argv = %#v, want %#v", got, want)
	}

	assertNoPlaceholders(t, rec)
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
