package processors

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
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

	// Filter repositories based on active profiles
	filteredRepositories := helpers.FilterByProfiles(repositoriesBlueprint.Repositories, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering repositories: %d total, %d matching active profiles %v",
		len(repositoriesBlueprint.Repositories), len(filteredRepositories), initConfig.Variables.Flags.Profiles)

	// Process the filtered repositories
	err = processRepositories(filteredRepositories, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	return nil
}

func processRepositories(repositories []types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers using the InitProviders function which handles embedded providers
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Process each repository
	for _, repo := range repositories {
		log.Infof("Processing repository %s", repo.Name)

		// Get provider for this repository
		provider, exists := system.GetProvider(repo.PackageManager)
		if !exists {
			return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
		}

		// Get repository config
		repoConfig := provider.Repository

		// Execute repository action steps
		var steps []types.ActionStep
		switch repo.Action {
		case "add":
			steps = repoConfig.Add.Steps
		case "remove":
			steps = repoConfig.Remove.Steps
		default:
			return fmt.Errorf("unsupported repository action: %s", repo.Action)
		}

		// Execute each step
		for _, step := range steps {
			var cmd types.Command

			switch step.Action {
			case "exec", "command": // Support both "exec" and "command" action types
				// Process template variables in args
				processedArgs := make([]string, len(step.Args))
				for i, arg := range step.Args {
					// Replace {{ .URL }} with the actual URL from the repository
					if arg == "{{ .URL }}" {
						processedArgs[i] = repo.URL
					} else {
						processedArgs[i] = arg
					}
				}

				cmd = types.Command{
					Exec:     step.Exec,
					Args:     processedArgs,
					Elevated: provider.Elevated,
				}
			case "write":
				if err := system.WriteToFile(step.Dest, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error writing file: %w", err)
				}
				continue
			case "copy":
				if err := system.CopyFile(step.Source, step.Dest, provider.Elevated, osInfo); err != nil {
					return fmt.Errorf("error copying file: %w", err)
				}
				continue
			default:
				return fmt.Errorf("unsupported repository action step: %s", step.Action)
			}

			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error executing repository step: %w", err)
			}
		}
	}

	// Run updates for all available providers
	available := system.GetAvailableProviders()
	for name, provider := range available {
		if provider.Commands.Update == "" {
			continue
		}

		log.Infof("Processing %s Updates", name)
		updateCmd := types.Command{
			Exec:     fmt.Sprintf("%s %s", provider.BinPath, provider.Commands.Update),
			Elevated: provider.Elevated,
		}

		if err := system.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error updating %s package lists: %v", name, err)
			continue
		}
	}

	return nil
}
