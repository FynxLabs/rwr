package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()
	viper.Set("rwr.configdir", configDir)
	t.Cleanup(func() { viper.Set("rwr.configdir", "") })
	return configDir
}

func TestBootstrapMarker_Sequence(t *testing.T) {
	configDir := withTempConfigDir(t)
	marker := filepath.Join(configDir, "bootstrap")

	// Bare run: base entries only.
	if err := Bootstrap(nil); err != nil {
		t.Fatalf("Bootstrap(nil): %v", err)
	}
	if !IsBootstrapped(nil) {
		t.Error("bare run should be bootstrapped for a bare follow-up")
	}
	if IsBootstrapped([]string{"work"}) {
		t.Error("base-only marker must not cover the work profile")
	}
	if IsBootstrapped([]string{"all"}) {
		t.Error("base-only marker must not cover 'all': gated entries were skipped")
	}

	// --profile work: re-runs, marker unions work in.
	if err := Bootstrap([]string{"work"}); err != nil {
		t.Fatalf("Bootstrap([work]): %v", err)
	}
	if !IsBootstrapped([]string{"work"}) {
		t.Error("work marker should cover the work profile")
	}
	if IsBootstrapped([]string{"personal"}) {
		t.Error("work marker must not cover the personal profile")
	}

	// --profile personal: union must keep work coverage. A machine alternating
	// between profiles must not re-run bootstrap on every switch.
	if err := Bootstrap([]string{"personal"}); err != nil {
		t.Fatalf("Bootstrap([personal]): %v", err)
	}
	if !IsBootstrapped([]string{"work"}) {
		t.Error("union lost work coverage after the personal run")
	}
	if !IsBootstrapped([]string{"personal"}) {
		t.Error("union lost personal coverage")
	}
	if !IsBootstrapped([]string{"work", "personal"}) {
		t.Error("union lost combined coverage")
	}
	if IsBootstrapped([]string{"gaming"}) {
		t.Error("union must not cover a profile neither run named")
	}

	// --profile all: covers everything, including gated entries a base-only
	// marker skipped.
	if err := Bootstrap([]string{"all"}); err != nil {
		t.Fatalf("Bootstrap([all]): %v", err)
	}
	if !IsBootstrapped([]string{"all"}) {
		t.Error("CoveredAll marker should cover an --profile all follow-up")
	}
	if !IsBootstrapped([]string{"gaming"}) {
		t.Error("CoveredAll marker should cover any profile")
	}

	// Legacy marker (empty file, older rwr): base entries only.
	if err := truncate(marker); err != nil {
		t.Fatal(err)
	}
	if !IsBootstrapped(nil) {
		t.Error("legacy marker should cover a bare follow-up")
	}
	if IsBootstrapped([]string{"work"}) {
		t.Error("legacy marker must not cover a profile request")
	}

	// Absent marker: nothing is covered, not even a bare run.
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if IsBootstrapped(nil) {
		t.Error("absent marker must not short-circuit a bare run")
	}
	if IsBootstrapped([]string{"work"}) {
		t.Error("absent marker must not short-circuit a profile run")
	}
}

func TestBootstrapMarker_MultipleRunsInSequence(t *testing.T) {
	configDir := withTempConfigDir(t)
	if err := Bootstrap(nil); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(configDir, "bootstrap")); err == nil {
		t.Logf("DEBUG marker after bare: %s", b)
	}
	// work then personal then work: every step covered, never a re-run trigger.
	for _, profiles := range [][]string{{"work"}, {"personal"}} {
		if IsBootstrapped(profiles) {
			t.Errorf("profiles %v: expected bootstrap to re-run (marker had no coverage yet)", profiles)
		}
		if err := Bootstrap(profiles); err != nil {
			t.Fatalf("Bootstrap(%v): %v", profiles, err)
		}
		if !IsBootstrapped(profiles) {
			t.Errorf("profiles %v: expected coverage after the run", profiles)
		}
	}
	// After work+personal, re-running either profile must be a skip.
	if !IsBootstrapped([]string{"work"}) || !IsBootstrapped([]string{"personal"}) {
		t.Error("alternating profiles lost coverage on the union")
	}
}

// truncate empties the marker file to simulate a legacy marker.
func truncate(path string) error {
	return os.WriteFile(path, []byte{}, 0o600)
}
