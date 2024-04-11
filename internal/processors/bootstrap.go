package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
)

func ProcessBootstrap(blueprintFile string, initConfig *types.InitConfig, osInfo types.OSInfo, forceBootstrap bool) error {
	if !forceBootstrap && helpers.IsBootstrapped() {
		log.Info("System is already bootstrapped. Skipping bootstrap process.")
		return nil
	}

	var bootstrapData types.BootstrapData

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &bootstrapData)
	if err != nil {
		log.Errorf("Error unmarshaling bootstrap blueprint: %v", err)
		return fmt.Errorf("error unmarshaling bootstrap blueprint: %w", err)
	}

	// Process packages
	err = ProcessPackagesFromFile(blueprintFile, osInfo)
	if err != nil {
		log.Errorf("Error processing packages: %v", err)
		return fmt.Errorf("error processing packages: %w", err)
	}

	// Process directories
	err = ProcessFiles(bootstrapData.Files)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return fmt.Errorf("error processing directories: %w", err)
	}

	// Process Git repositories
	err = ProcessGitRepositories(bootstrapData.Git)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return fmt.Errorf("error processing Git repositories: %w", err)
	}

	// Process services
	err = ProcessServices(bootstrapData.Services)
	if err != nil {
		log.Errorf("Error processing services: %v", err)
		return fmt.Errorf("error processing services: %w", err)
	}

	// Process users/groups
	err = ProcessUsers(bootstrapData.Users)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	err = ProcessGroups(bootstrapData.Groups)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return fmt.Errorf("error processing groups: %w", err)
	}

	// Set the bootstrap file
	err = helpers.Bootstrap()
	if err != nil {
		log.Errorf("Error setting bootstrap file: %v", err)
		return fmt.Errorf("error setting bootstrap file: %w", err)
	}

	log.Info("Bootstrap process completed successfully.")
	return nil
}
