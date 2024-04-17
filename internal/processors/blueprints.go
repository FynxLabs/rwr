package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"
)

func GetBlueprints(initConfig *types.InitConfig) (string, error) {
	// Check if GitOptions is provided in the init configuration
	if initConfig.Init.Git != nil {
		gitOpts := initConfig.Init.Git

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

	if initConfig.Init.Order != nil {
		for _, item := range initConfig.Init.Order {
			if str, ok := item.(string); ok {
				runOrder = append(runOrder, str)
			} else if subOrder, ok := item.(map[string]interface{}); ok {
				for processor := range subOrder {
					runOrder = append(runOrder, processor)
				}
			}
		}
	} else {
		runOrder = append(runOrder, "packageManagers", "repositories", "packages", "files", "templates", "configuration", "services")
	}

	log.Debugf("Blueprint run order: %v", runOrder)
	return runOrder, nil
}

func GetBlueprintFileOrder(blueprintDir string, order []interface{}, runOnlyListed bool, initConfig *types.InitConfig) (map[string][]string, error) {
	fileOrder := make(map[string][]string)

	for _, item := range order {
		if str, ok := item.(string); ok {
			// Check if the item is a directory
			processorDir := filepath.Join(blueprintDir, str)
			if _, err := os.Stat(processorDir); err == nil {
				// If the item is a directory, scan for files and add them to the fileOrder
				err := filepath.Walk(processorDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
						relPath, err := filepath.Rel(blueprintDir, path)
						if err != nil {
							return err
						}
						fileOrder[str] = append(fileOrder[str], relPath)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else {
				// If the item is a file, add it directly to the fileOrder
				fileOrder[filepath.Dir(str)] = append(fileOrder[filepath.Dir(str)], str)
			}
		} else if subOrder, ok := item.(map[string]interface{}); ok {
			for processor, files := range subOrder {
				if filesArr, ok := files.([]interface{}); ok {
					for _, file := range filesArr {
						if fileStr, ok := file.(string); ok {
							fileOrder[processor] = append(fileOrder[processor], filepath.Join(processor, fileStr))
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
			if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
				relPath, err := filepath.Rel(blueprintDir, path)
				if err != nil {
					return err
				}
				processor := filepath.Dir(relPath)
				if _, ok := fileOrder[processor]; !ok {
					fileOrder[processor] = []string{relPath}
				} else if !helpers.Contains(fileOrder[processor], relPath) {
					fileOrder[processor] = append(fileOrder[processor], relPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	log.Debugf("Blueprint file order: %v", fileOrder)
	return fileOrder, nil
}
