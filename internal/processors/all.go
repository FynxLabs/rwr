package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
	"path/filepath"
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

			switch processorName {
			case "repositories":
				err = ProcessRepositoriesFromFile(blueprintFile, osInfo)
			case "packages":
				err = ProcessPackagesFromFile(blueprintFile, osInfo)
			case "files":
				err = ProcessFilesFromFile(blueprintFile)
			case "services":
				err = ProcessServicesFromFile(blueprintFile)
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
