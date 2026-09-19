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
// the sorted set of profiles the bootstrap covered, and whether the run
// included every profile-scoped entry (--profile all). A later run re-runs
// bootstrap only when the current run names something the marker does not
// cover.
type BootstrapMarker struct {
	// Profiles is the sorted set of named profiles the bootstrap ran with.
	// An empty list means the run applied base entries only.
	Profiles []string `json:"profiles"`
	// CoveredAll records a run with the "all" profile: every gated entry
	// applied, so no profile name can be missing from coverage.
	CoveredAll bool `json:"covered_all,omitempty"`
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

// readBootstrapMarker returns the recorded marker, or a zero marker when the
// file is absent or unreadable. A zero marker covers base entries only.
func readBootstrapMarker() BootstrapMarker {
	bootstrapFile, err := bootstrapMarkerPath()
	if err != nil {
		return BootstrapMarker{}
	}
	data, err := os.ReadFile(bootstrapFile) // #nosec G304 -- path is operator-supplied config input
	if err != nil {
		return BootstrapMarker{}
	}

	var marker BootstrapMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		// Unreadable marker: treat it as a bare marker from an older version.
		// It covers base entries; any profile request re-runs bootstrap.
		log.Debugf("Bootstrap marker is not valid JSON (%v); treating as base-only", err)
		return BootstrapMarker{}
	}
	return marker
}

// covers reports whether the marker covers the given active profile set.
func (m BootstrapMarker) covers(activeProfiles []string) bool {
	if m.CoveredAll {
		return true
	}
	for _, profile := range activeProfiles {
		if profile == "all" {
			// "all" activates every gated entry; only a CoveredAll marker
			// guarantees none was skipped.
			return false
		}
		if !slices.Contains(m.Profiles, profile) {
			return false
		}
	}
	return true
}

// IsBootstrapped reports whether bootstrap can be skipped for this run: a
// marker exists and covers every active profile. Legacy markers (empty file or
// no JSON) cover base entries only, so any profile request re-runs bootstrap.
// An absent marker never short-circuits, even for a bare run.
func IsBootstrapped(activeProfiles []string) bool {
	bootstrapFile, err := bootstrapMarkerPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(bootstrapFile); err != nil {
		return false
	}
	return readBootstrapMarker().covers(activeProfiles)
}

// Bootstrap updates the marker file in the rwr config directory, recording the
// union of the profiles already covered and the ones this run covered. Union,
// not replace: alternating between two profiles must not make each run forget
// the other's coverage and re-run bootstrap on every switch.
func Bootstrap(activeProfiles []string) error {
	bootstrapFile, err := bootstrapMarkerPath()
	if err != nil {
		return err
	}

	previous := readBootstrapMarker()
	covered := slices.Concat(previous.Profiles, activeProfiles)
	covered = slices.DeleteFunc(covered, func(p string) bool { return p == "all" })
	slices.Sort(covered)
	covered = slices.Compact(covered)

	next := BootstrapMarker{Profiles: covered, CoveredAll: previous.CoveredAll || slices.Contains(activeProfiles, "all")}
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	return os.WriteFile(bootstrapFile, data, 0o600) // #nosec G304 -- path is operator-supplied config input
}
