package processors

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// Inspect preparation before credential resolution or any mutation. Rendering is
// lenient because ordinary credentials intentionally resolve after bootstrap.
func validateBootstrapPreparation(path string, config *types.InitConfig) error {
	if err := types.ValidatePackageManagers(config.PackageManagers); err != nil {
		return err
	}
	top, _, err := decodeTopLevel(path, config)
	if err != nil {
		return err
	}
	data, err := json.Marshal(top["packageManagers"])
	if err != nil {
		return err
	}
	var managers []types.PackageManagerInfo
	if err := json.Unmarshal(data, &managers); err != nil {
		return err
	}
	return types.ValidatePackageManagers(managers)
}

// Successful preparation steps survive a later bootstrap failure. A failed or
// interrupted script must itself be safe to retry; no marker can undo its effects.
func processBootstrapScripts(scripts []types.Script, osInfo *types.OSInfo, config *types.InitConfig, dir string) error {
	for _, script := range scripts {
		if system.IsDryRun() {
			if err := processScripts([]types.Script{script}, osInfo, config, dir); err != nil {
				return err
			}
			continue
		}
		identity, err := json.Marshal(struct {
			Dir    string
			Script types.Script
		}{dir, script})
		if err != nil {
			return err
		}
		if script.Source != "" {
			source, err := os.ReadFile(filepath.Join(dir, script.Source, script.Name)) // #nosec G304 -- operator-supplied script source
			if err != nil {
				return err
			}
			identity = append(identity, source...)
		}
		configDir := viper.GetString("rwr.configdir")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			configDir = filepath.Join(home, ".config", "rwr")
		}
		markerDir := filepath.Join(configDir, "bootstrap-steps")
		if err := os.MkdirAll(markerDir, 0700); err != nil {
			return err
		}
		marker := filepath.Join(markerDir, fmt.Sprintf("%x", sha256.Sum256(identity)))
		if _, err := os.Stat(marker); err == nil { // #nosec G703 -- operator-selected RWR config directory; filename is a SHA-256 hex digest, never blueprint path text
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := processScripts([]types.Script{script}, osInfo, config, dir); err != nil {
			return err
		}
		if err := persistBootstrapStep(marker); err != nil {
			return err
		}
	}
	return nil
}

func persistBootstrapStep(path string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".completed-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name()) //nolint:errcheck
	if err := file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path) // #nosec G703 -- both paths are in the operator-selected marker directory; destination basename is a SHA-256 digest
}
