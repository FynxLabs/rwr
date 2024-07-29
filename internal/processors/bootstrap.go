package processors

import (
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessBootstrap(blueprintFile string, initConfig *types.InitConfig, osInfo *types.OSInfo) error {
	if !initConfig.Variables.Flags.ForceBootstrap && helpers.IsBootstrapped() {
		log.Info("System is already bootstrapped. Skipping bootstrap process.")
		return nil
	}

	log.Info("Starting bootstrap processor...")

	var bootstrapData types.BootstrapData
	var blueprintData []byte
	var err error

	// Resolve variables in the blueprint file if templates are enabled
	blueprintData, err = os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return err
	}

	blueprintData, err = helpers.ResolveTemplate(blueprintData, initConfig.Variables)
	if err != nil {
		log.Errorf("Error resolving variables in bootstrap file: %v", err)
		return err
	}

	blueprintDir := filepath.Dir(blueprintFile)

	format := initConfig.Init.Format
	if blueprintFile != "" {
		format = filepath.Ext(blueprintFile)
	}

	// Unmarshal the blueprint data
	log.Debugf("Unmarshaling bootstrap data from %s", blueprintFile)
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &bootstrapData)
	if err != nil {
		log.Errorf("Error unmarshaling bootstrap blueprint: %v", err)
		return err
	}

	// Process packages
	log.Debugf("Processing packages from %s", blueprintFile)
	packagesData := &types.PackagesData{
		Packages: bootstrapData.Packages,
	}
	err = ProcessPackages(nil, packagesData, format, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing packages: %v", err)
		return err
	}

	// Process directories
	log.Debugf("Processing directories from %s", blueprintFile)
	err = processDirectories(bootstrapData.Directories, blueprintDir, initConfig)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process Files
	log.Debugf("Processing files from %s", blueprintFile)
	err = processFiles(bootstrapData.Files, blueprintDir, initConfig)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process SSH
	log.Debugf("Processing files from %s", blueprintFile)
	err = processSSHKeys(bootstrapData.SSHKeys, osInfo, initConfig)
	if err != nil {
		log.Errorf("Error processing directories: %v", err)
		return err
	}

	// Process Git repositories
	log.Debugf("Processing Git repositories from %s", blueprintFile)
	err = processGitRepositories(bootstrapData.Git)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return err
	}

	// Process services
	log.Debugf("Processing services from %s", blueprintFile)
	err = processServices(bootstrapData.Services, initConfig)
	if err != nil {
		log.Errorf("Error processing services: %v", err)
		return err
	}

	// Process users/groups
	log.Debugf("Processing users/groups from %s", blueprintFile)
	err = processUsers(bootstrapData.Users, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	// Process groups
	log.Debugf("Processing groups from %s", blueprintFile)
	err = processGroups(bootstrapData.Groups, initConfig)
	if err != nil {
		log.Errorf("Error processing groups: %v", err)
		return err
	}

	// Set the bootstrap file
	log.Debugf("Setting bootstrap fileProcessDirectories")
	err = helpers.Bootstrap()
	if err != nil {
		log.Errorf("Error setting bootstrap file: %v", err)
		return err
	}

	log.Info("Bootstrap processor completed successfully.")
	return nil
}
