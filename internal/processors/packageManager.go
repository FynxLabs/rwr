package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// packageManagerTempDir is the per-run private directory install/remove steps
// stage into via {{ .TempDir }}.
var (
	pmTempDirOnce sync.Once
	pmTempDirPath string
)

func packageManagerTempDir() string {
	pmTempDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "rwr-pm-")
		if err != nil {
			// Surfaced at use: the write into the unusable directory fails
			// with a real error naming the path.
			log.Errorf("could not create the package manager staging directory: %v", err)
			pmTempDirPath = filepath.Join(os.TempDir(), "rwr-pm-unavailable")
			return
		}
		pmTempDirPath = dir
	})
	return pmTempDirPath
}

// ProcessPackageManagers installs package managers and their common dependencies
// (OpenSSL, build essentials) before installing each requested package manager.
func ProcessPackageManagers(packageManagers []types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers if needed
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	log.Infof("Installing package manager common dependencies")

	// Install OpenSSL
	log.Infof("Installing OpenSSL")
	if err := system.InstallOpenSSL(osInfo, initConfig); err != nil {
		return fmt.Errorf("error installing OpenSSL: %v", err)
	}

	// Install build essentials
	log.Infof("Installing build essentials")
	if err := system.InstallBuildEssentials(osInfo, initConfig); err != nil {
		return fmt.Errorf("error installing build essentials: %v", err)
	}

	// Process each package manager
	for _, pm := range packageManagers {
		log.Debugf("Processing package manager: %s (action: %s)", pm.Name, pm.Action)
		// The definition-level lookup, not GetProvider: GetProvider reports a
		// provider whose binary is missing as unavailable, and a missing
		// binary is exactly the state an install starts from. With GetProvider
		// here, install steps could never run at all — absent binary errored,
		// present binary skipped as already installed.
		provider, exists := system.GetProviderDefinition(pm.Name)
		if !exists {
			return fmt.Errorf("no provider definition for package manager %s", pm.Name)
		}

		// Check if already installed
		if pm.Action == "install" && system.FindTool(provider.Detection.Binary).Exists {
			log.Infof("%s is already installed", pm.Name)
			continue
		}

		// Get steps based on action
		var steps []types.ActionStep
		switch pm.Action {
		case "install":
			steps = provider.Install.Steps
		case "remove":
			steps = provider.Remove.Steps
		default:
			return fmt.Errorf("unsupported package manager action: %s", pm.Action)
		}

		// Execute each step
		for _, rawStep := range steps {
			// Install steps historically staged at fixed, world-known /tmp
			// names (/tmp/brew-install.sh, a git clone into /tmp/yay). Any
			// local user can pre-create such a path — or rewrite it between
			// the download and the elevated step that executes it — which is
			// root code execution. {{ .TempDir }} renders to a per-run 0700
			// directory other users cannot reach.
			step, err := renderActionStep(rawStep, map[string]any{"TempDir": packageManagerTempDir()})
			if err != nil {
				return fmt.Errorf("error rendering %s step for %s: %w", pm.Action, pm.Name, err)
			}

			var cmd types.Command

			switch step.Action {
			case "command":
				cmd = types.Command{
					Exec:     step.Exec,
					Args:     step.Args,
					Elevated: provider.Elevated,
					AsUser:   pm.AsUser,
				}
			case "download":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would download file: %s -> %s", step.Source, step.Dest)
					continue
				}
				if err := system.DownloadFile(step.Source, step.Dest, provider.Elevated); err != nil {
					return fmt.Errorf("error downloading file: %w", err)
				}
				continue
			case "write":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would write file: %s", step.Dest)
					continue
				}
				if err := system.WriteToFile(step.Dest, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error writing file: %w", err)
				}
				continue
			default:
				return fmt.Errorf("unsupported package manager action step: %s", step.Action)
			}

			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error executing package manager step: %w", err)
			}
		}
	}

	return nil
}
