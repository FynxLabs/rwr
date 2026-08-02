package processors

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// The ledger contract (failures.go): a processor keeps going when one item
// fails, records the failure, and All() turns the ledger into the exit code.
// packages, git, ssh_keys and configuration honored it; files, services,
// repositories and fonts aborted the whole run on the first bad item instead.
// These tests pin the contract for the four late joiners: the bad item is
// recorded, the good item still happens, the processor returns nil.

func TestProcessFiles_BadFileDoesNotStopTheRest(t *testing.T) {
	resetFailures()
	defer resetFailures()

	tempDir := t.TempDir()
	blueprintDir := filepath.Join(tempDir, "blueprints")

	config := &types.InitConfig{
		Init:      types.Init{Location: blueprintDir, Format: "yaml"},
		Variables: types.Variables{Flags: types.Flags{Debug: true}},
	}

	// The first file's source does not exist; the second is a plain create.
	blueprintData := `
files:
  - name: "missing.conf"
    action: "copy"
    source: "does-not-exist"
    target: "` + tempDir + `/"
  - name: "good.conf"
    action: "create"
    content: "key = value"
    target: "` + tempDir + `/"
`

	if err := ProcessFiles([]byte(blueprintData), blueprintDir, "yaml", &types.OSInfo{}, config); err != nil {
		t.Fatalf("ProcessFiles = %v, want nil: item failures belong in the ledger", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "good.conf")); err != nil {
		t.Errorf("the file after the failing one was not processed: %v", err)
	}
	if got := failureCount(); got != 1 {
		t.Errorf("failureCount() = %d, want 1", got)
	}
}

func TestProcessServices_BadServiceDoesNotStopTheRest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("asserts the linux/darwin service command path")
	}
	resetFailures()
	defer resetFailures()

	rec := exectest.New()
	rec.Err = errors.New("boom")
	defer system.SetExecutor(rec)()

	services := []types.Service{
		{Name: "first", Action: "enable"},
		{Name: "second", Action: "enable"},
	}

	if err := processServices(services, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processServices = %v, want nil: item failures belong in the ledger", err)
	}

	if len(rec.Calls) != 2 {
		t.Errorf("recorded %d calls, want 2 — the second service must still be attempted: %v", len(rec.Calls), rec.Calls)
	}
	if got := failureCount(); got != 2 {
		t.Errorf("failureCount() = %d, want 2", got)
	}
}

func TestProcessRepositories_BadRepositoryDoesNotStopTheRest(t *testing.T) {
	resetFailures()
	defer resetFailures()

	sourcesDir, keysDir := repoDirs(t)
	provider := providerForTest(t, "flatpak", sourcesDir, keysDir)

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"flatpak": provider})()

	if err := processRepositories([]types.Repository{
		{Name: "broken", PackageManager: "no-such-manager", Action: "add", URL: "https://example.com"},
		{Name: "flathub", PackageManager: "flatpak", Action: "add", URL: "https://dl.flathub.org/repo/flathub.flatpakrepo"},
	}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories = %v, want nil: item failures belong in the ledger", err)
	}

	if calls := rec.Find("flatpak"); len(calls) == 0 {
		t.Errorf("the repository after the failing one was not processed: %v", rec.Calls)
	}
	if got := failureCount(); got != 1 {
		t.Errorf("failureCount() = %d, want 1", got)
	}
}

func TestProcessFonts_UnreachableReleaseLookupIsALedgerFailure(t *testing.T) {
	resetFailures()
	defer resetFailures()

	orig := nerdFontRepoAPI
	nerdFontRepoAPI = "http://127.0.0.1:0/unreachable"
	defer func() { nerdFontRepoAPI = orig }()

	blueprintData := `
fonts:
  - name: "FiraCode"
    action: "install"
`

	config := &types.InitConfig{
		Variables: types.Variables{Flags: types.Flags{Debug: true}},
	}

	if err := ProcessFonts([]byte(blueprintData), "", "yaml", &types.OSInfo{}, config); err != nil {
		t.Fatalf("ProcessFonts = %v, want nil: the failed lookup belongs in the ledger", err)
	}
	if got := failureCount(); got != 1 {
		t.Errorf("failureCount() = %d, want 1", got)
	}
}
