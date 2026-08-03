package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

func bootstrapInit(location string) *types.InitConfig {
	init := &types.InitConfig{}
	init.Init.Location = location
	init.Init.Format = "yaml"
	return init
}

// A tree without a bootstrap file is an explicit error naming the candidate
// filenames, not a silent no-op.
func TestRunBootstrap_MissingFileNamesCandidates(t *testing.T) {
	err := RunBootstrap(bootstrapInit(t.TempDir()), &types.OSInfo{})
	if err == nil {
		t.Fatal("missing bootstrap file did not error")
	}
	for _, want := range []string{"bootstrap.yaml", "bootstrap.toml"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name candidate %s", err, want)
		}
	}
}

// Asking for bootstrap by name bypasses the run-once marker; `rwr all`'s
// gating (ProcessBootstrap honoring the marker) is untouched.
func TestRunBootstrap_BypassesMarker(t *testing.T) {
	configDir := t.TempDir()
	viper.Set("rwr.configdir", configDir)
	defer viper.Set("rwr.configdir", "")

	// Marker present: the system says "already bootstrapped".
	if err := os.WriteFile(filepath.Join(configDir, "bootstrap"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	markerBefore, _ := os.Stat(filepath.Join(configDir, "bootstrap"))

	tree := t.TempDir()
	if err := os.WriteFile(filepath.Join(tree, "bootstrap.yaml"), []byte("packages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	init := bootstrapInit(tree)
	if err := RunBootstrap(init, &types.OSInfo{}); err != nil {
		t.Fatalf("standalone bootstrap with marker present: %v", err)
	}

	// The flag restore must not leak the forced value into later runs.
	if init.Variables.Flags.ForceBootstrap {
		t.Fatal("ForceBootstrap leaked past the standalone run")
	}

	// The marker was refreshed, not removed.
	markerAfter, err := os.Stat(filepath.Join(configDir, "bootstrap"))
	if err != nil {
		t.Fatalf("marker gone after standalone run: %v", err)
	}
	if markerAfter.ModTime().Before(markerBefore.ModTime()) {
		t.Fatal("marker not refreshed")
	}

	// And ProcessBootstrap without force still skips — all's gating intact.
	if err := ProcessBootstrap(filepath.Join(tree, "bootstrap.yaml"), init, &types.OSInfo{}); err != nil {
		t.Fatalf("gated ProcessBootstrap errored instead of skipping: %v", err)
	}
}
