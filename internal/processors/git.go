package processors

import (
	"fmt"
	"os"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessGitRepositories(blueprintData []byte, format string, initConfig *types.InitConfig) error {
	var gitData types.GitData
	var err error

	log.Debugf("Processing Git repositories from blueprint")

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, format, &gitData)
	if err != nil {
		return fmt.Errorf("error unmarshaling Git repository blueprint: %w", err)
	}

	// Process the Git repositories
	err = processGitRepositories(gitData.Repos, initConfig)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return fmt.Errorf("error processing Git repositories: %w", err)
	}

	return nil
}

func processGitRepositories(gitRepos []types.Git, initConfig *types.InitConfig) error {
	for _, repo := range gitRepos {
		gitOpts := types.GitOptions{
			URL:     repo.URL,
			Private: repo.Private,
			Target:  repo.Path,
			Branch:  repo.Branch,
		}

		_, err := os.Stat(gitOpts.Target)
		if err == nil {
			// Repository already exists, check and update remote URL
			log.Infof("Git repository %s already exists at %s", repo.Name, gitOpts.Target)
			err = helpers.CheckAndUpdateRemoteURL(gitOpts.Target, gitOpts.URL)
			if err != nil {
				log.Errorf("Error checking/updating remote URL for %s: %v", repo.Name, err)
				return err
			}
		} else if os.IsNotExist(err) {
			// Repository doesn't exist, clone it
			err = helpers.HandleGitClone(gitOpts, initConfig)
			if err != nil {
				log.Errorf("Error cloning Git repository %s: %v", repo.Name, err)
				return err
			}
			log.Infof("Git repository %s cloned successfully", repo.Name)
		} else {
			// Some other error occurred
			return fmt.Errorf("error checking Git repository %s: %w", repo.Name, err)
		}
	}
	return nil
}

// func cloneGitRepository(repo types.Git, initConfig *types.InitConfig) error {
// 	gitOpts := types.GitOptions{
// 		URL:     repo.URL,
// 		Private: repo.Private,
// 		Target:  repo.Path,
// 		Branch:  repo.Branch,
// 	}

// 	_, err := os.Stat(gitOpts.Target)
// 	if err == nil {
// 		// Repository already exists
// 		log.Infof("Git repository %s already exists at %s", repo.Name, gitOpts.Target)
// 		return nil
// 	} else if !os.IsNotExist(err) {
// 		// Some other error occurred
// 		return fmt.Errorf("error checking Git repository %s: %w", repo.Name, err)
// 	}

// 	err = helpers.HandleGitClone(gitOpts, initConfig)
// 	if err != nil {
// 		return fmt.Errorf("error cloning Git repository %s: %w", repo.Name, err)
// 	}

// 	return nil
// }
