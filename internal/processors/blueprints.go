package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func GetBlueprintsLocation(initConfig *types.InitConfig) (string, error) {
	// Check if GitOptions is provided in the init configuration
	if initConfig.Blueprint.Git != nil {
		gitOpts := initConfig.Blueprint.Git

		// Check if the target directory already exists
		if _, err := os.Stat(gitOpts.Target); os.IsNotExist(err) {
			// If the target directory doesn't exist, clone the Git repository
			err := helpers.HandleGitClone(*gitOpts)
			if err != nil {
				return "", fmt.Errorf("error cloning Git repository: %v", err)
			}
		} else if gitOpts.Update {
			// If the target directory exists and an update is requested, perform a git pull
			err := helpers.HandleGitPull(*gitOpts)
			if err != nil {
				return "", fmt.Errorf("error updating Git repository: %v", err)
			}
		}

		// Check if init.(yaml|json|toml) exists at the root of the target directory
		initFilePath := filepath.Join(gitOpts.Target, "init.yaml")
		if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
			initFilePath = filepath.Join(gitOpts.Target, "init.json")
			if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
				initFilePath = filepath.Join(gitOpts.Target, "init.toml")
				if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
					log.Errorf("Init file not found in the blueprints location")
					return "", fmt.Errorf("init file not found in the blueprints location")
				}
			}
		}

		log.Infof("Using init file: %s", initFilePath)
		return gitOpts.Target, nil
	}

	// If GitOptions is not provided, use the local path
	localPath := initConfig.Blueprint.Location

	// Check if init.(yaml|json|toml) exists at the root of the local path
	initFilePath := filepath.Join(localPath, "init.yaml")
	if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
		initFilePath = filepath.Join(localPath, "init.json")
		if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
			initFilePath = filepath.Join(localPath, "init.toml")
			if _, err := os.Stat(initFilePath); os.IsNotExist(err) {
				log.Errorf("Init file not found in the blueprints location")
				return "", fmt.Errorf("init file not found in the blueprints location")
			}
		}
	}

	log.Infof("Using init file: %s", initFilePath)
	return localPath, nil
}

func GetBlueprintRunOrder(initConfig *types.InitConfig) ([]string, error) {
	var runOrder []string
	for _, item := range initConfig.Blueprint.Order {
		if str, ok := item.(string); ok {
			runOrder = append(runOrder, str)
		} else if subOrder, ok := item.(map[string]interface{}); ok {
			for processor := range subOrder {
				runOrder = append(runOrder, processor)
			}
		}
	}
	return runOrder, nil
}

func GetBlueprintFileOrder(blueprintDir string, order []interface{}, runOnlyListed bool, initConfig *types.InitConfig) ([]string, error) {
	var fileOrder []string
	for _, item := range order {
		if str, ok := item.(string); ok {
			fileOrder = append(fileOrder, str)
		} else if subOrder, ok := item.(map[string]interface{}); ok {
			for processor, files := range subOrder {
				if filesArr, ok := files.([]interface{}); ok {
					for _, file := range filesArr {
						if fileStr, ok := file.(string); ok {
							fileOrder = append(fileOrder, filepath.Join(processor, fileStr))
						}
					}
				}
			}
		}
	}

	if !runOnlyListed {
		// Add remaining files in the blueprint directories if runOnlyListed is false
		err := filepath.Walk(blueprintDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Blueprint.Format {
				relPath, err := filepath.Rel(blueprintDir, path)
				if err != nil {
					return err
				}
				if !helpers.Contains(fileOrder, relPath) {
					fileOrder = append(fileOrder, relPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return fileOrder, nil
}
