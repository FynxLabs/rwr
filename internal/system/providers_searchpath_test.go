package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

const testProviderTOML = `[provider]
name = "%s"
elevated = true

[provider.detection]
binary = "definitely-not-a-real-binary"
distributions = ["linux"]
`

func writeProviderFile(t *testing.T, dir, name string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(dir, name+".toml")
	contents := strings.Replace(testProviderTOML, "%s", name, 1)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing provider %s: %v", path, err)
	}
	// Written then chmodded because os.WriteFile applies the process umask, which
	// would strip the group/world bits this test depends on.
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// TestGetProvidersPathIgnoresWorkingDirectory guards the search path against
// ./providers: a provider defines commands that run elevated, so picking them up
// from wherever rwr happens to be run is remote code execution by cd.
func TestGetProvidersPathIgnoresWorkingDirectory(t *testing.T) {
	cwd := t.TempDir()
	evil := filepath.Join(cwd, "providers")
	if err := os.MkdirAll(evil, 0o755); err != nil {
		t.Fatalf("creating providers dir: %v", err)
	}
	writeProviderFile(t, evil, "evil", 0o600)

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})

	path, err := GetProvidersPath()
	if err == nil && path == evil {
		t.Fatalf("GetProvidersPath returned the working directory: %s", path)
	}
}

// TestLoadProvidersSkipsWritableFiles checks that a definition anyone can edit is
// not loaded, since its commands run as root.
func TestLoadProvidersSkipsWritableFiles(t *testing.T) {
	restore := SetProvidersForTest(map[string]*types.Provider{})
	defer restore()

	dir := t.TempDir()
	writeProviderFile(t, dir, "safe", 0o644)
	writeProviderFile(t, dir, "world-writable", 0o666)
	writeProviderFile(t, dir, "group-writable", 0o664)

	if err := LoadProviders(dir); err != nil {
		t.Fatalf("LoadProviders: %v", err)
	}

	providersMu.Lock()
	defer providersMu.Unlock()

	if _, ok := providers["safe"]; !ok {
		t.Error("expected the 0644 provider to load")
	}
	for _, name := range []string{"world-writable", "group-writable"} {
		if _, ok := providers[name]; ok {
			t.Errorf("provider %q is writable by others and should have been skipped", name)
		}
	}
}
