package processors

import (
	"fmt"
	"os"
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
	pmTempDirErr  error
)

// packageManagerTempDir returns the run's private staging directory, or the
// error that prevented creating it. A failure used to fall back to the
// predictable <tmp>/rwr-pm-unavailable - a fixed, world-known name any local
// user can pre-create, which is exactly the class {{ .TempDir }} exists to
// eliminate.
func packageManagerTempDir() (string, error) {
	pmTempDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "rwr-pm-")
		if err != nil {
			pmTempDirErr = fmt.Errorf("could not create the package manager staging directory: %w", err)
			return
		}
		pmTempDirPath = dir
	})
	return pmTempDirPath, pmTempDirErr
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
		// here, install steps could never run at all - absent binary errored,
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
			// local user can pre-create such a path - or rewrite it between
			// the download and the elevated step that executes it - which is
			// root code execution. {{ .TempDir }} renders to a per-run 0700
			// directory other users cannot reach.
			tempDir, err := packageManagerTempDir()
			if err != nil {
				return err
			}
			step, err := renderActionStep(rawStep, map[string]any{"TempDir": tempDir})
			if err != nil {
				return fmt.Errorf("error rendering %s step for %s: %w", pm.Action, pm.Name, err)
			}

			var cmd types.Command

			switch step.Action {
			case "command":
				// Resolve through FindTool, not the raw process PATH: an
				// install sequence runs the binary it just installed (rustup
				// puts cargo in ~/.cargo/bin, brew lands in /opt/homebrew/bin)
				// and none of those are on the PATH rwr started with.
				execPath := step.Exec
				if tool := system.FindTool(step.Exec); tool.Exists {
					execPath = tool.Bin
				}
				cmd = types.Command{
					Exec:     execPath,
					Args:     step.Args,
					Elevated: provider.Elevated,
					AsUser:   pm.AsUser,
					// The provider's environment applies to its own install
					// steps too: brew's NONINTERACTIVE is what keeps the
					// official install.sh from stopping at "Press RETURN".
					Variables: provider.Environment,
				}
			case "download":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would download file: %s -> %s", step.Source, step.Dest)
					continue
				}
				// step.Sha256 is rendered by renderActionStep like every other
				// field; dropping it here silently waived the digest check the
				// provider author asked for (repository.go threads it through).
				if err := system.DownloadFileWithChecksum(step.Source, step.Dest, provider.Elevated, step.Sha256); err != nil {
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
				if step.Optional {
					log.Warnf("Optional %s step for %s failed (ignored): %v", pm.Action, pm.Name, err)
					continue
				}
				return fmt.Errorf("error executing package manager step: %w", err)
			}
		}
	}

	return nil
}
