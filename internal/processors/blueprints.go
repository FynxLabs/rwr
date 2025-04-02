package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5"
)

func GetBlueprints(initConfig *types.InitConfig) (string, error) {
	// Check if GitOptions is provided in the init configuration
	if initConfig.Init.Git != nil {
		gitOpts := initConfig.Init.Git

		// Clean up any existing non-git directory
		if _, err := os.Stat(gitOpts.Target); err == nil {
			// Try to open as git repo to verify it's actually a git repository
			_, err := git.PlainOpen(gitOpts.Target)
			if err != nil {
				// Directory exists but is not a git repo - remove it and clone fresh
				log.Debugf("Directory exists but is not a git repository, removing: %s", gitOpts.Target)
				if err := os.RemoveAll(gitOpts.Target); err != nil {
					return "", fmt.Errorf("error removing existing directory: %v", err)
				}
			}
		}

		// Now either clone fresh or update existing
		_, err := git.PlainOpen(gitOpts.Target)
		if err != nil {
			// Repository doesn't exist or was removed - clone it
			log.Debugf("Cloning blueprint repository to %s", gitOpts.Target)
			if err := os.MkdirAll(filepath.Dir(gitOpts.Target), 0755); err != nil {
				return "", fmt.Errorf("error creating parent directory: %v", err)
			}
			err = helpers.HandleGitClone(*gitOpts, initConfig)
			if err != nil {
				return "", fmt.Errorf("error cloning blueprint repository: %v", err)
			}
			log.Debugf("Blueprint repository cloned successfully")
		} else {
			// Repository exists and is valid - update it
			log.Debugf("Updating existing blueprint repository at %s", gitOpts.Target)
			err = helpers.CheckAndUpdateRemoteURL(gitOpts.Target, gitOpts.URL)
			if err != nil {
				return "", fmt.Errorf("error checking/updating remote URL: %v", err)
			}

			if gitOpts.Update {
				err = helpers.HandleGitPull(*gitOpts, initConfig)
				if err != nil {
					return "", fmt.Errorf("error updating blueprint repository: %v", err)
				}
			}
			log.Debugf("Blueprint repository updated successfully")
		}

		// Verify the blueprints directory exists and has content
		filesInfo, err := os.ReadDir(gitOpts.Target)
		if err != nil {
			return "", fmt.Errorf("error reading blueprints directory: %v", err)
		}
		if len(filesInfo) == 0 {
			return "", fmt.Errorf("blueprints directory is empty: %s", gitOpts.Target)
		}

		log.Debugf("Using blueprint location: %s", gitOpts.Target)
		return gitOpts.Target, nil
	}

	// If GitOptions is not provided, use the default location from initConfig
	location := initConfig.Init.Location
	log.Debugf("Using default blueprint location: %s", location)
	return location, nil
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
		runOrder = append(runOrder, "packageManagers", "repositories", "packages", "ssh_keys", "files", "fonts", "services", "git", "scripts", "configuration")
	}

	log.Debugf("Blueprint run order: %v", runOrder)
	return runOrder, nil
}

func GetBlueprintFileOrder(blueprintDir string, order []interface{}, runOnlyListed bool, initConfig *types.InitConfig) (map[string][]string, error) {
	fileOrder := make(map[string][]string)

	log.Debugf("Getting blueprint file order from directory: %s", blueprintDir)

	// Helper function to extract processor type from path
	getProcessorType := func(path string) string {
		parts := strings.Split(path, string(os.PathSeparator))
		// Look for known processor types in the path
		for _, part := range parts {
			switch part {
			case "packages", "repositories", "files", "services", "users",
				"git", "scripts", "ssh_keys", "fonts", "configuration":
				return part
			}
		}
		return filepath.Dir(path)
	}

	// Process ordered items first
	for _, item := range order {
		if str, ok := item.(string); ok {
			fullPath := filepath.Join(blueprintDir, str)

			if info, err := os.Stat(fullPath); err == nil {
				if info.IsDir() {
					// Process directory
					err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
							relPath, err := filepath.Rel(blueprintDir, path)
							if err != nil {
								return err
							}
							processor := getProcessorType(relPath)
							fileOrder[processor] = append(fileOrder[processor], relPath)
							log.Debugf("Added file to processor %s: %s", processor, relPath)
						}
						return nil
					})
					if err != nil {
						return nil, err
					}
				} else {
					// Single file
					relPath, err := filepath.Rel(blueprintDir, fullPath)
					if err != nil {
						return nil, err
					}
					processor := getProcessorType(relPath)
					fileOrder[processor] = append(fileOrder[processor], relPath)
					log.Debugf("Added single file to processor %s: %s", processor, relPath)
				}
			}
		}
	}

	// If not runOnlyListed, scan for additional files
	if !runOnlyListed {
		err := filepath.Walk(blueprintDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == "."+initConfig.Init.Format {
				relPath, err := filepath.Rel(blueprintDir, path)
				if err != nil {
					return err
				}
				processor := getProcessorType(relPath)
				if _, exists := fileOrder[processor]; !exists {
					fileOrder[processor] = []string{relPath}
				} else if !helpers.Contains(fileOrder[processor], relPath) {
					fileOrder[processor] = append(fileOrder[processor], relPath)
				}
				log.Debugf("Added additional file to processor %s: %s", processor, relPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Log final order
	for processor, files := range fileOrder {
		log.Debugf("Processor %s files:", processor)
		for _, file := range files {
			log.Debugf("  - %s", file)
		}
	}

	return fileOrder, nil
}
