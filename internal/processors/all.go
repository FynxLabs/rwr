package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/types"
	"path/filepath"
)

func All(initConfig *types.InitConfig, osInfo *types.OSInfo, runOrder []string) error {
	var err error
	var blueprintRunOrder []string

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
		forceBootstrap := viper.GetBool("force-bootstrap")
		err = ProcessBootstrap(filepath.Join(initConfig.Init.Location, "bootstrap.yaml"), initConfig, osInfo, forceBootstrap)
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
						err = ProcessRepositoriesFromData(resolvedBlueprint, osInfo, initConfig)
					} else {
						err = ProcessRepositoriesFromFile(blueprintFile, osInfo, initConfig)
					}
				case "packages":
					log.Infof("Processing packages")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessPackagesFromData(resolvedBlueprint, osInfo, initConfig)
					} else {
						err = ProcessPackagesFromFile(blueprintFile, osInfo, initConfig)
					}
				case "files":
					log.Infof("Processing files")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessFilesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessFilesFromFile(blueprintFile)
					}
				case "services":
					log.Infof("Processing services")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessServicesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessServicesFromFile(blueprintFile, initConfig)
					}
				case "templates":
					log.Infof("Processing templates")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessTemplatesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessTemplatesFromFile(blueprintFile, initConfig)
					}
				case "users":
					log.Infof("Processing users")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessUsersFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessUsersFromFile(blueprintFile, initConfig)
					}
				case "git":
					log.Infof("Processing Git repositories")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessGitRepositoriesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessGitRepositoriesFromFile(blueprintFile)
					}
				case "scripts":
					log.Infof("Processing scripts")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessScriptsFromData(resolvedBlueprint, osInfo, initConfig)
					} else {
						err = ProcessScriptsFromFile(blueprintFile, osInfo, initConfig)
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
