package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
)

func GetBlueprints(initConfig *types.InitConfig) (string, error) {
	// Check if GitOptions is provided in the init configuration
	if initConfig.Blueprint.Git != nil {
		gitOpts := initConfig.Blueprint.Git

		// Check if the target directory already exists
		if _, err := os.Stat(gitOpts.Target); os.IsNotExist(err) {
			// If the target directory doesn't exist, clone the Git repository
			log.Debugf("Directory %s does not exist. Cloning Git repository", gitOpts.Target)
			err := helpers.HandleGitClone(*gitOpts)
			if err != nil {
				return "", fmt.Errorf("error cloning Git repository: %v", err)
			}
		} else if gitOpts.Update {
			// If the target directory exists and an update is requested, perform a git pull
			log.Debugf("Directory %s exists. Updating Git repository", gitOpts.Target)
			err := helpers.HandleGitPull(*gitOpts)
			if err != nil {
				return "", fmt.Errorf("error updating Git repository: %v", err)
			}
		}

		log.Debugf("Git repository cloned/updated successfully")
		return "", nil
	}

	// If GitOptions is not provided, return the blueprint directory
	log.Debugf("Blueprints are not managed by Git.")
	return "", nil
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
