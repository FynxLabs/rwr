package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func All(initConfig *types.InitConfig, osInfo *types.OSInfo, runOrder []string) error {
	var err error
	var blueprintRunOrder []string

	log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)

	if runOrder != nil {
		blueprintRunOrder = append([]string(nil), runOrder...)
	} else {
		blueprintRunOrder, err = GetBlueprintRunOrder(initConfig)
		if err != nil {
			return fmt.Errorf("error getting blueprint run order: %w", err)
		}
	}

	// Run the bootstrap processor first if it exists
	bootstrapFile := filepath.Join(initConfig.Init.Location, "bootstrap.yaml")
	if helpers.FileExists(bootstrapFile) {
		err = ProcessBootstrap(bootstrapFile, initConfig, osInfo)
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
				blueprintDir := filepath.Dir(blueprintFile)
				log.Debugf("Processing %s from file: %s", processor, blueprintFile)

				format := initConfig.Init.Format
				if blueprintFile != "" {
					format = filepath.Ext(blueprintFile)
				}

				blueprintData, err := os.ReadFile(blueprintFile)
				if err != nil {
					return fmt.Errorf("error reading blueprint file %s: %w", blueprintFile, err)
				}

				resolvedBlueprint, err := helpers.ResolveTemplate(blueprintData, initConfig.Variables)
				if err != nil {
					return fmt.Errorf("error resolving variables in %s: %w", processor, err)
				}

				switch processor {
				case "repositories":
					log.Infof("Processing repositories")
					err = ProcessRepositories(resolvedBlueprint, format, osInfo, initConfig)
				case "packages":
					log.Infof("Processing packages")
					err = ProcessPackages(resolvedBlueprint, nil, format, osInfo, initConfig)
				case "files":
					log.Infof("Processing files")
					err = ProcessFiles(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
				case "services":
					log.Infof("Processing services")
					err = ProcessServices(resolvedBlueprint, format, osInfo, initConfig)
				case "users":
					log.Infof("Processing users")
					err = ProcessUsers(resolvedBlueprint, format, initConfig)
				case "git":
					log.Infof("Processing git repositories")
					err = ProcessGitRepositories(resolvedBlueprint, format, initConfig)
				case "scripts":
					log.Infof("Processing scripts")
					err = ProcessScripts(resolvedBlueprint, blueprintDir, format, osInfo, initConfig)
				case "ssh_keys":
					log.Infof("Processing ssh keys")
					err = ProcessSSHKeys(resolvedBlueprint, format, osInfo, initConfig)
				case "fonts":
					log.Info("Processing fonts")
					err = ProcessFonts(blueprintData, blueprintDir, format, osInfo, initConfig)
				case "configuration":
					log.Infof("Processing configurations")
					err = ProcessConfiguration(resolvedBlueprint, blueprintDir, format, initConfig)
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
