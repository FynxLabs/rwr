package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
)

func All(initConfig *types.InitConfig) error {

	osInfo := DetectOS()
	var err error

	// Process repositories
	err = ProcessRepositories(initConfig.Repositories, osInfo)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	// Process package managers
	err = ProcessPackageManagers(initConfig.PackageManagers, osInfo)
	if err != nil {
		return fmt.Errorf("error processing package managers: %w", err)
	}

	// Process packages
	err = ProcessPackages(initConfig.Packages, osInfo)
	if err != nil {
		return fmt.Errorf("error processing packages: %w", err)
	}

	// Process services
	err = ProcessServices(initConfig.Services, osInfo)
	if err != nil {
		return fmt.Errorf("error processing services: %w", err)
	}

	// Process files
	err = ProcessFiles(initConfig.Files)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Process directories
	err = ProcessDirectories(initConfig.Directories)
	if err != nil {
		return fmt.Errorf("error processing directories: %w", err)
	}

	log.Info("Initialization completed")

}
