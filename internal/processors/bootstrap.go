package processors

import (
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
)

func ProcessBootstrap(blueprintFile string, initConfig *types.InitConfig, osInfo *types.OSInfo, forceBootstrap bool) error {
	if !forceBootstrap && helpers.IsBootstrapped() {
		log.Info("System is already bootstrapped. Skipping bootstrap process.")
		return nil
	}

	log.Info("Starting bootstrap processor...")

	var bootstrapData types.BootstrapData
	var blueprintData []byte

	// Resolve variables in the blueprint file if templates are enabled
	if initConfig.Init.TemplatesEnabled {
		var err error
		blueprintData, err = RenderTemplate(blueprintFile, initConfig.Variables)
		if err != nil {
			log.Errorf("Error resolving variables in bootstrap file: %v", err)
			return err
		}
	} else {
		// Read the blueprint file without resolving variables
		var err error
		blueprintData, err = os.ReadFile(blueprintFile)
		if err != nil {
			log.Errorf("Error reading blueprint file: %v", err)
			return err
		}
	}

	// Unmarshal the blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &bootstrapData)
	if err != nil {
		log.Errorf("Error unmarshaling bootstrap blueprint: %v", err)
		return err
	}

	// Process packages
	err = ProcessPackagesFromFile(blueprintFile, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing packages: %v", err)
		return err
	}

	// Process directories
	err = ProcessFiles(bootstrapData.Files)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process Git repositories
	err = ProcessGitRepositories(bootstrapData.Git)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return err
	}

	// Process services
	err = ProcessServices(bootstrapData.Services, initConfig)
	if err != nil {
		log.Errorf("Error processing services: %v", err)
		return err
	}

	// Process users/groups
	err = ProcessUsers(bootstrapData.Users, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	err = ProcessGroups(bootstrapData.Groups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	// Set the bootstrap file
	err = helpers.Bootstrap()
	if err != nil {
		log.Errorf("Error setting bootstrap file: %v", err)
		return err
	}

	log.Info("Bootstrap processor completed successfully.")
	return nil
}
