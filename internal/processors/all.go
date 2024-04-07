package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
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
		switch processorName {
		case "repositories":
			err = ProcessRepositories(initConfig.Repositories, osInfo)
		case "packages":
			err = ProcessPackages(initConfig.Packages, osInfo)
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
