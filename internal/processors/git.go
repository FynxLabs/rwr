package processors

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Process imports and merge imported git repos
	blueprintDir := initConfig.Init.Location
	allRepos, err := processGitImports(gitData.Repos, blueprintDir, format)
	if err != nil {
		return fmt.Errorf("error processing git imports: %w", err)
	}
	gitData.Repos = allRepos

	// Filter Git repositories based on active profiles
	filteredRepos := helpers.FilterByProfiles(gitData.Repos, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering Git repositories: %d total, %d matching active profiles %v",
		len(gitData.Repos), len(filteredRepos), initConfig.Variables.Flags.Profiles)

	// Process the filtered Git repositories
	err = processGitRepositories(filteredRepos, initConfig)
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
				log.Warnf("Error checking/updating remote URL for %s: %v", repo.Name, err)
				// Continue with other repositories instead of returning
				continue
			}

			// Pull latest changes
			err = helpers.HandleGitPull(gitOpts, initConfig)
			if err != nil {
				log.Warnf("Error pulling latest changes for %s: %v", repo.Name, err)
				// Continue with other repositories instead of returning
				continue
			}
			log.Infof("Git repository %s updated successfully", repo.Name)
		} else if os.IsNotExist(err) {
			// Repository doesn't exist, clone it
			err = helpers.HandleGitClone(gitOpts, initConfig)
			if err != nil {
				log.Warnf("Error cloning Git repository %s: %v", repo.Name, err)
				// Continue with other repositories instead of returning
				continue
			}
			log.Infof("Git repository %s cloned successfully", repo.Name)
		} else {
			// Some other error occurred
			log.Warnf("Error checking Git repository %s: %v", repo.Name, err)
			// Continue with other repositories instead of returning
			continue
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

func processGitImports(repos []types.Git, blueprintDir string, format string) ([]types.Git, error) {
	allRepos := make([]types.Git, 0)
	visited := make(map[string]bool)

	for _, repo := range repos {
		if repo.Import != "" {
			log.Debugf("Processing git import: %s", repo.Import)

			importPath := filepath.Join(blueprintDir, repo.Import)
			absPath, err := filepath.Abs(importPath)
			if err != nil {
				return nil, fmt.Errorf("error resolving import path %s: %w", importPath, err)
			}

			if visited[absPath] {
				log.Warnf("Circular import detected, skipping: %s", absPath)
				continue
			}
			visited[absPath] = true

			importData, err := os.ReadFile(importPath)
			if err != nil {
				return nil, fmt.Errorf("error reading import file %s: %w", importPath, err)
			}

			fileFormat := format
			if fileFormat == "" {
				ext := filepath.Ext(importPath)
				fileFormat = ext
			}

			var importedGitData types.GitData
			if err := helpers.UnmarshalBlueprint(importData, fileFormat, &importedGitData); err != nil {
				return nil, fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
			}

			allRepos = append(allRepos, importedGitData.Repos...)
			log.Debugf("Imported %d git repos from %s", len(importedGitData.Repos), repo.Import)
		} else {
			allRepos = append(allRepos, repo)
		}
	}

	return allRepos, nil
}
