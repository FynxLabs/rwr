package processors

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

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
		provider, exists := system.GetProvider(pm.Name)
		if !exists {
			// GetProvider will have already logged detailed error info
			return fmt.Errorf("package manager %s is not available - check debug logs for details", pm.Name)
		}

		// Check if already installed
		if pm.Action == "install" && system.FindTool(provider.Detection.Binary).Exists {
			log.Infof("%s is already installed", pm.Name)
			continue
		}

		// Get steps based on action
		var steps []types.ActionStep
		if pm.Action == "install" {
			steps = provider.Install.Steps
		} else if pm.Action == "remove" {
			steps = provider.Remove.Steps
		} else {
			return fmt.Errorf("unsupported package manager action: %s", pm.Action)
		}

		// Execute each step
		for _, step := range steps {
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
				if err := system.DownloadFile(step.Source, step.Dest, provider.Elevated); err != nil {
					return fmt.Errorf("error downloading file: %w", err)
				}
				continue
			case "write":
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
