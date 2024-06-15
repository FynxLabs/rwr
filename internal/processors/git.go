package processors

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
)

func ProcessGitRepositoriesFromFile(blueprintFile string, blueprintDir string) error {
	var gitData types.GitData
	var gitRepos []types.Git

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		log.Errorf("Error reading blueprint file: %v", err)
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &gitData)
	if err != nil {
		log.Errorf("Error unmarshaling Git repository blueprint: %v", err)
		return fmt.Errorf("error unmarshaling Git repository blueprint: %w", err)
	}

	gitRepos = gitData.Repos

	// Process the Git repositories
	err = ProcessGitRepositories(gitRepos)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return fmt.Errorf("error processing Git repositories: %w", err)
	}

	return nil
}

func ProcessGitRepositoriesFromData(blueprintData []byte, blueprintDir string, initConfig *types.InitConfig) error {
	var gitData types.GitData
	var gitRepos []types.Git

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &gitData)
	if err != nil {
		log.Errorf("Error unmarshaling Git repository blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling Git repository blueprint data: %w", err)
	}

	gitRepos = gitData.Repos

	// Process the Git repositories
	err = ProcessGitRepositories(gitRepos)
	if err != nil {
		log.Errorf("Error processing Git repositories: %v", err)
		return fmt.Errorf("error processing Git repositories: %w", err)
	}

	return nil
}

func ProcessGitRepositories(gitRepos []types.Git) error {
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
		log.Errorf("Error checking Git repository %s: %v", repo.Name, err)
		return fmt.Errorf("error checking Git repository %s: %w", repo.Name, err)
	}

	err = helpers.HandleGitClone(gitOpts)
	if err != nil {
		log.Errorf("Error cloning Git repository %s: %v", repo.Name, err)
		return fmt.Errorf("error cloning Git repository %s: %w", repo.Name, err)
	}

	return nil
}
