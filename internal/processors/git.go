package processors

import (
	"fmt"
	"github.com/fynxlabs/rwr/internal/types"
	"os"

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
	err = processGitRepositories(gitData.Repos)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return fmt.Errorf("error processing Git repositories: %w", err)
	}

	return nil
}

func processGitRepositories(gitRepos []types.Git) error {
	for _, repo := range gitRepos {
		if repo.Action == "clone" {
			err := cloneGitRepository(repo)
			if err != nil {
				log.Errorf("Error processing Git repository %s: %v", repo.Name, err)
				return fmt.Errorf("error processing Git repository %s: %w", repo.Name, err)
			}
			log.Infof("Git repository %s cloned successfully", repo.Name)
		} else {
			log.Errorf("Unsupported action for Git repository %s: %s", repo.Name, repo.Action)
			return fmt.Errorf("unsupported action for Git repository %s: %s", repo.Name, repo.Action)
		}
	}
	return nil
}

func cloneGitRepository(repo types.Git) error {
	gitOpts := types.GitOptions{
		URL:     repo.URL,
		Private: repo.Private,
		Target:  repo.Path,
		Branch:  repo.Branch,
	}

	_, err := os.Stat(gitOpts.Target)
	if err == nil {
		// Repository already exists
		log.Infof("Git repository %s already exists at %s", repo.Name, gitOpts.Target)
		return nil
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("error checking Git repository %s: %w", repo.Name, err)
	}

	err = helpers.HandleGitClone(gitOpts)
	if err != nil {
		return fmt.Errorf("error cloning Git repository %s: %w", repo.Name, err)
	}

	return nil
}
