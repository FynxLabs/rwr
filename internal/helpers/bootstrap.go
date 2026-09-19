package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"charm.land/log/v2"
	"github.com/spf13/viper"
)

// BootstrapMarker is the on-disk record of a completed bootstrap. It records
// the sorted set of profiles the bootstrap covered: a later run naming a
// profile outside that set re-runs bootstrap so the newly named entries apply.
type BootstrapMarker struct {
	// Profiles is the sorted set of active profiles the bootstrap ran with.
	// An empty list means the run applied base entries only.
	Profiles []string `json:"profiles"`
}

// bootstrapMarkerPath returns the marker file's path, creating nothing.
func bootstrapMarkerPath() (string, error) {
	configDir := viper.GetString("rwr.configdir")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "rwr")
	}
	return filepath.Join(configDir, "bootstrap"), nil
}

// bootstrapNeeded reports whether bootstrap must run: the marker is absent, or
// the current run names profiles the recorded bootstrap did not cover.
//
// A marker with no profile list (written by older rwr versions) covers only
// base entries, so any profile request re-runs bootstrap.
func bootstrapNeeded(activeProfiles []string) (bool, string) {
	bootstrapFile, err := bootstrapMarkerPath()
	if err != nil {
		return true, ""
	}
	data, err := os.ReadFile(bootstrapFile) // #nosec G304 -- path is operator-supplied config input
	if err != nil {
		return true, ""
	}

	var marker BootstrapMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		// Unreadable marker: treat it as a bare marker from an older version.
		// It covers base entries; any profile request re-runs bootstrap.
		log.Debugf("Bootstrap marker is not valid JSON (%v); treating as base-only", err)
		marker = BootstrapMarker{}
	}

	for _, profile := range activeProfiles {
		if profile == "all" {
			continue // covered by anything: a re-run adds nothing the marker lacks
		}
		if !slices.Contains(marker.Profiles, profile) {
			return true, profile
		}
	}
	return false, ""
}

// IsBootstrapped reports whether bootstrap can be skipped for this run: a
// marker exists and covers every active profile. Legacy markers (empty file or
// no JSON) cover base entries only, so any profile request re-runs bootstrap.
func IsBootstrapped(activeProfiles []string) bool {
	needed, _ := bootstrapNeeded(activeProfiles)
	return !needed
}

// Bootstrap creates a marker file in the rwr config directory recording that
// bootstrap completed for the given active profiles.
func Bootstrap(activeProfiles []string) error {
	bootstrapFile, err := bootstrapMarkerPath()
	if err != nil {
		return err
	}

	covered := slices.Clone(activeProfiles)
	slices.Sort(covered)
	covered = slices.Compact(covered)

	data, err := json.Marshal(BootstrapMarker{Profiles: covered})
	if err != nil {
		return err
	}
	return os.WriteFile(bootstrapFile, data, 0o600) // #nosec G304 -- path is operator-supplied config input
}
