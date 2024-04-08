package processors

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
	"text/template"
)

func All(initConfig *types.InitConfig, runOrder []string) error {
	osInfo := DetectOS()
	var err error
	var blueprintRunOrder []string

	if runOrder != nil {
		blueprintRunOrder = runOrder
	} else {
		blueprintRunOrder, err = GetBlueprintRunOrder(initConfig)
		if err != nil {
			return fmt.Errorf("error getting blueprint run order: %w", err)
		}
	}

	for _, processor := range blueprintRunOrder {
		processorName := processor
		fileOrder, err := GetBlueprintFileOrder(initConfig.Blueprint.Location, initConfig.Blueprint.Order, initConfig.Blueprint.RunOnlyListed, initConfig)
		if err != nil {
			return fmt.Errorf("error getting blueprint file order: %w", err)
		}

		for _, file := range fileOrder {
			blueprintFile := filepath.Join(initConfig.Blueprint.Location, file)
			var resolvedBlueprint []byte
			// Resolve variables in the blueprint file
			if initConfig.Blueprint.TemplatesEnabled {
				resolvedBlueprint, err = resolveVariables(blueprintFile, initConfig.Variables)
				if err != nil {
					return fmt.Errorf("error resolving variables in blueprint file: %w", err)
				}
			}

			switch processorName {
			case "repositories":
				if initConfig.Blueprint.TemplatesEnabled {
					err = ProcessRepositoriesFromData(resolvedBlueprint, initConfig, osInfo)
				} else {
					err = ProcessRepositoriesFromFile(blueprintFile, osInfo)
				}
			case "packages":
				if initConfig.Blueprint.TemplatesEnabled {
					err = ProcessPackagesFromData(resolvedBlueprint, initConfig, osInfo)
				} else {
					err = ProcessPackagesFromFile(blueprintFile, osInfo)
				}
			case "files":
				if initConfig.Blueprint.TemplatesEnabled {
					err = ProcessFilesFromData(resolvedBlueprint, initConfig)
				} else {
					err = ProcessFilesFromFile(blueprintFile)
				}
			case "services":
				if initConfig.Blueprint.TemplatesEnabled {
					err = ProcessServicesFromData(resolvedBlueprint, initConfig)
				} else {
					err = ProcessServicesFromFile(blueprintFile)
				}
			default:
				log.Warnf("Unknown processor: %s", processorName)
				continue
			}

			if err != nil {
				return fmt.Errorf("error processing %s: %w", processorName, err)
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
