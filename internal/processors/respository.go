package processors

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessRepositories(blueprintData []byte, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var repositoriesBlueprint types.RepositoriesData
	var err error

	log.Debugf("Processing repositories from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &repositoriesBlueprint)
	if err != nil {
		return fmt.Errorf("error unmarshaling repository blueprint: %w", err)
	}

	// Process the repositories
	err = processRepositories(repositoriesBlueprint.Repositories, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	return nil
}

func processRepositories(repositories []types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	// Process each repository
	for _, repo := range repositories {
		log.Infof("Processing repository %s", repo.Name)

		// Get provider for this repository
		provider, exists := providers.GetProvider(repo.PackageManager)
		if !exists {
			return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
		}

		// Get repository config
		repoConfig := provider.Repository

		// Execute repository action steps
		var steps []providers.ActionStep
		if repo.Action == "add" {
			steps = repoConfig.Add.Steps
		} else if repo.Action == "remove" {
			steps = repoConfig.Remove.Steps
		} else {
			return fmt.Errorf("unsupported repository action: %s", repo.Action)
		}

		// Execute each step
		for _, step := range steps {
			var cmd types.Command

			switch step.Action {
			case "exec":
				cmd = types.Command{
					Exec:     step.Exec,
					Args:     step.Args,
					Elevated: provider.Elevated,
				}
			case "write":
				if err := helpers.WriteToFile(step.Dest, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error writing file: %w", err)
				}
				continue
			case "copy":
				if err := helpers.CopyFile(step.Source, step.Dest, provider.Elevated, osInfo); err != nil {
					return fmt.Errorf("error copying file: %w", err)
				}
				continue
			default:
				return fmt.Errorf("unsupported repository action step: %s", step.Action)
			}

			if err := helpers.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error executing repository step: %w", err)
			}
		}
	}

	// Run updates for all available providers
	available := providers.GetAvailableProviders()
	for name, provider := range available {
		if provider.Commands.Update == "" {
			continue
		}

		log.Infof("Processing %s Updates", name)
		updateCmd := types.Command{
			Exec:     fmt.Sprintf("%s %s", provider.BinPath, provider.Commands.Update),
			Elevated: provider.Elevated,
		}

		if err := helpers.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error updating %s package lists: %v", name, err)
			continue
		}
	}

	return nil
}
