package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
)

func All(initConfig *types.InitConfig) error {
	osInfo := DetectOS()
	var err error

	blueprintRunOrder, err := GetBlueprintRunOrder(initConfig)
	if err != nil {
		return fmt.Errorf("error getting blueprint run order: %w", err)
	}

	for _, processor := range blueprintRunOrder {
		processorName := processor.(string)
		switch processorName {
		case "repositories":
			err = ProcessRepositories(initConfig.Repositories, osInfo)
		case "packages":
			err = ProcessPackages(initConfig, osInfo)
		case "files":
			err = ProcessFiles(initConfig.Files)
		//case "templates":
		//	err = ProcessTemplates(initConfig.Templates)
		//case "configuration":
		//	err = ProcessConfigurations(initConfig.Configuration)
		case "services":
			err = ProcessServices(initConfig.Services, osInfo)
		default:
			log.Warnf("Unknown processor: %s", processorName)
			continue
		}

		if err != nil {
			return fmt.Errorf("error processing %s: %w", processorName, err)
		}
	}

	log.Info("Initialization completed")
	return nil
}
