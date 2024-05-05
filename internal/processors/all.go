package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/types"
	"path/filepath"
)

func All(initConfig *types.InitConfig, osInfo *types.OSInfo, runOrder []string) error {
	var err error
	var blueprintRunOrder []string

	log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)

	if runOrder != nil {
		blueprintRunOrder = append(runOrder)
	} else {
		blueprintRunOrder, err = GetBlueprintRunOrder(initConfig)
		if err != nil {
			return fmt.Errorf("error getting blueprint run order: %w", err)
		}
	}

	// Run the bootstrap processor first if it exists
	if helpers.FileExists(filepath.Join(initConfig.Init.Location, "bootstrap.yaml")) {
		err = ProcessBootstrap(filepath.Join(initConfig.Init.Location, "bootstrap.yaml"), initConfig, osInfo)
		if err != nil {
			return fmt.Errorf("error processing bootstrap: %w", err)
		}
	}

	// Process package managers
	if initConfig.PackageManagers != nil {
		log.Debugf("Processing package managers")
		err = ProcessPackageManagers(initConfig.PackageManagers, osInfo, initConfig)
		if err != nil {
			return fmt.Errorf("error processing package managers: %w", err)
		}
	}

	fileOrder, err := GetBlueprintFileOrder(initConfig.Init.Location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
	if err != nil {
		return fmt.Errorf("error getting blueprint file order: %w", err)
	}

	for _, processor := range blueprintRunOrder {
		if files, ok := fileOrder[processor]; ok {
			for _, file := range files {
				blueprintFile := filepath.Join(initConfig.Init.Location, file)
				blueprintDir := filepath.Dir(blueprintFile)
				log.Debugf("Processing %s from file: %s", processor, blueprintFile)
				var resolvedBlueprint []byte
				// Resolve variables in the blueprint file
				if initConfig.Init.TemplatesEnabled {
					resolvedBlueprint, err := RenderTemplate(blueprintFile, initConfig.Variables)
					log.Debugf("Resolved blueprint: %s", resolvedBlueprint)
					if err != nil {
						log.Errorf("error resolving variables in %s: %v", processor, err)
						return err
					}
				}

				switch processor {
				case "repositories":
					log.Infof("Processing repositories")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessRepositoriesFromData(resolvedBlueprint, blueprintDir, osInfo, initConfig)
					} else {
						err = ProcessRepositoriesFromFile(blueprintFile, blueprintDir, osInfo, initConfig)
					}
				case "packages":
					log.Infof("Processing packages")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessPackagesFromData(resolvedBlueprint, blueprintDir, osInfo, initConfig)
					} else {
						err = ProcessPackagesFromFile(blueprintFile, blueprintDir, osInfo, initConfig)
					}
				case "files":
					log.Infof("Processing files")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessFilesFromData(resolvedBlueprint, blueprintDir, initConfig)
					} else {
						err = ProcessFilesFromFile(blueprintFile, blueprintDir, initConfig)
					}
				case "services":
					log.Infof("Processing services")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessServicesFromData(resolvedBlueprint, blueprintDir, initConfig)
					} else {
						err = ProcessServicesFromFile(blueprintFile, blueprintDir, initConfig)
					}
				case "templates":
					log.Infof("Processing templates")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessTemplatesFromData(resolvedBlueprint, blueprintDir, initConfig)
					} else {
						err = ProcessTemplatesFromFile(blueprintFile, blueprintDir, initConfig)
					}
				case "users":
					log.Infof("Processing users")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessUsersFromData(resolvedBlueprint, blueprintDir, initConfig)
					} else {
						err = ProcessUsersFromFile(blueprintFile, blueprintDir, initConfig)
					}
				case "git":
					log.Infof("Processing Git repositories")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessGitRepositoriesFromData(resolvedBlueprint, blueprintDir, initConfig)
					} else {
						err = ProcessGitRepositoriesFromFile(blueprintFile, blueprintDir)
					}
				case "scripts":
					log.Infof("Processing scripts")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessScriptsFromData(resolvedBlueprint, blueprintDir, osInfo, initConfig)
					} else {
						err = ProcessScriptsFromFile(blueprintFile, blueprintDir, osInfo, initConfig)
					}
				default:
					log.Warnf("Unknown processor: %s", processor)
					continue
				}

				if err != nil {
					return fmt.Errorf("error processing %s: %w", processor, err)
				}
			}
		}
	}

	log.Infof("Cleaning up package managers")
	err = helpers.CleanPackageManagers(osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error cleaning package managers: %w", err)
	}

	log.Info("RWR Run Complete!")
	return nil
}
