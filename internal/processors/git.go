package processors

import (
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessGitRepositories clones or updates Git repositories defined in blueprint data,
// with support for profile filtering and import resolution.
func ProcessGitRepositories(blueprintData []byte, format string, initConfig *types.InitConfig) error {
	var gitData types.GitData
	var err error

	log.Debugf("Processing Git repositories from blueprint")

	// Unmarshal the blueprint data
	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeGit,
		helpers.TreeSchemaVersion(initConfig), &gitData)
	if err != nil {
		return fmt.Errorf("error unmarshaling Git repository blueprint: %w", err)
	}

	// Process imports and merge imported git repos
	blueprintDir := initConfig.Init.Location
	allRepos, err := processGitImports(gitData.Repos, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
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
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would clone/update git repository: %s -> %s", repo.URL, repo.Path)
			continue
		}
		gitOpts := types.GitOptions{
			URL:     repo.URL,
			Private: repo.Private,
			Target:  system.ExpandPath(repo.Path),
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

func processGitImports(items []types.Git, blueprintDir string, format string, treeVersion int) ([]types.Git, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Git) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Git, error) {
			var d types.GitData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeGit, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Repos, nil
		}, format)
}
