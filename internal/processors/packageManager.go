package processors

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessPackageManagers(packageManagers []types.PackageManagerInfo, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	log.Infof("Installing package manager common dependencies")

	// Install OpenSSL
	log.Infof("Installing OpenSSL")
	if err := helpers.InstallOpenSSL(osInfo, initConfig); err != nil {
		return fmt.Errorf("error installing OpenSSL: %v", err)
	}

	// Install build essentials
	log.Infof("Installing build essentials")
	if err := helpers.InstallBuildEssentials(osInfo, initConfig); err != nil {
		return fmt.Errorf("error installing build essentials: %v", err)
	}

	// Process each package manager
	for _, pm := range packageManagers {
		provider, exists := providers.GetProvider(pm.Name)
		if !exists {
			return fmt.Errorf("unsupported package manager: %s", pm.Name)
		}

		// Check if already installed
		if pm.Action == "install" && helpers.FindTool(provider.Detection.Binary).Exists {
			log.Infof("%s is already installed", pm.Name)
			continue
		}

		// Get steps based on action
		var steps []providers.ActionStep
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
				if err := helpers.DownloadFile(step.Source, step.Dest, provider.Elevated); err != nil {
					return fmt.Errorf("error downloading file: %w", err)
				}
				continue
			case "write":
				if err := helpers.WriteToFile(step.Dest, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error writing file: %w", err)
				}
				continue
			default:
				return fmt.Errorf("unsupported package manager action step: %s", step.Action)
			}

			if err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error executing package manager step: %w", err)
			}
		}
	}

	return nil
}
