package processors

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
	"text/template"
)

func All(initConfig *types.InitConfig, runOrder []string) error {
	osInfo := helpers.DetectOS()
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
		err = ProcessPackageManagers(initConfig.PackageManagers, osInfo)
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
					resolvedBlueprint, err = resolveVariables(blueprintFile, initConfig.Variables)
					log.Debugf("Resolved blueprint: %s", resolvedBlueprint)
					if err != nil {
						return fmt.Errorf("error resolving variables in blueprint file: %w", err)
					}
				}

				switch processor {
				case "repositories":
					log.Debugf("Processing repositories")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessRepositoriesFromData(resolvedBlueprint, initConfig, osInfo)
					} else {
						err = ProcessRepositoriesFromFile(blueprintFile, osInfo)
					}
				case "packages":
					log.Debugf("Processing packages")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessPackagesFromData(resolvedBlueprint, initConfig, osInfo)
					} else {
						err = ProcessPackagesFromFile(blueprintFile, osInfo)
					}
				case "files":
					log.Debugf("Processing files")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessFilesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessFilesFromFile(blueprintFile)
					}
				case "services":
					log.Debugf("Processing services")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessServicesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessServicesFromFile(blueprintFile)
					}
				case "templates":
					log.Debugf("Processing templates")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessTemplatesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessTemplatesFromFile(blueprintFile)
					}
				case "users":
					log.Debugf("Processing users")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessUsersFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessUsersFromFile(blueprintFile)
					}
				case "git":
					log.Debugf("Processing Git repositories")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessGitRepositoriesFromData(resolvedBlueprint, initConfig)
					} else {
						err = ProcessGitRepositoriesFromFile(blueprintFile)
					}
				case "scripts":
					log.Debugf("Processing scripts")
					if initConfig.Init.TemplatesEnabled {
						err = ProcessScriptsFromData(resolvedBlueprint, initConfig, osInfo)
					} else {
						err = ProcessScriptsFromFile(blueprintFile, osInfo)
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

	log.Info("Initialization completed")
	return nil
}

func resolveVariables(blueprintFile string, variables map[string]interface{}) ([]byte, error) {
	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		return nil, fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Create a new template
	t, err := template.New(filepath.Base(blueprintFile)).Parse(string(blueprintData))
	if err != nil {
		return nil, fmt.Errorf("error parsing blueprint template: %w", err)
	}

	// Execute the template
	var resolvedData bytes.Buffer
	err = t.Execute(&resolvedData, variables)
	if err != nil {
		return nil, fmt.Errorf("error executing blueprint template: %w", err)
	}

	return resolvedData.Bytes(), nil
}
