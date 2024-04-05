package helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/viper"
)

func GetBlueprintsLocation(update bool) (string, error) {
	localPath := viper.GetString("repository.blueprints.localPath")
	remoteStoreType := viper.GetString("repository.blueprints.remoteStoreType")
	remoteStoreURL := viper.GetString("repository.blueprints.remoteStoreURL")

	if remoteStoreType == "git" {
		// Check if the local path already exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			// If the local path doesn't exist, clone the Git repository
			_, err := git.PlainClone(localPath, false, &git.CloneOptions{
				URL: remoteStoreURL,
			})
			if err != nil {
				log.Errorf("Error cloning Git repository: %v", err)
				return "", fmt.Errorf("error cloning Git repository: %v", err)
			}
			log.Infof("Git repository cloned to: %s", localPath)
		} else if update {
			// If the local path exists and an update is requested, perform a git pull
			repo, err := git.PlainOpen(localPath)
			if err != nil {
				log.Errorf("Error opening Git repository: %v", err)
				return "", fmt.Errorf("error opening Git repository: %v", err)
			}

			worktree, err := repo.Worktree()
			if err != nil {
				log.Errorf("Error getting worktree: %v", err)
				return "", fmt.Errorf("error getting worktree: %v", err)
			}

			err = worktree.Pull(&git.PullOptions{})
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				log.Errorf("Error pulling changes from Git repository: %v", err)
				return "", fmt.Errorf("error pulling changes from Git repository: %v", err)
			}
			log.Infof("Git repository updated: %s", localPath)
		}
	}

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
