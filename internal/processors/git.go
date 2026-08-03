package processors

import (
	"fmt"
	"os"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessGitRepositories clones or updates Git repositories defined in blueprint data,
// with support for profile filtering and import resolution.
func ProcessGitRepositories(blueprintData []byte, blueprintDir string, format string, initConfig *types.InitConfig) error {
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
	track := newProgress(types.BlueprintTypeGit)
	track.expect("", len(gitRepos))
	for _, repo := range gitRepos {
		started := time.Now()
		if system.IsDryRun() {
			log.Infof("[DRY-RUN] Would clone/update git repository: %s -> %s", repo.URL, repo.Path)
			track.item("", repo.Name, repo.Action, types.StatusPlanned, "dry-run", 0)
			continue
		}
		// The action field was decoded and never read, so `action: pull` cloned and
		// `action: banana` was accepted. Empty and "clone" keep the original
		// behavior — clone when missing, pull when present.
		switch repo.Action {
		case "", types.GitActionClone, types.GitActionPull:
		default:
			recordFailure("git", repo.Name,
				fmt.Errorf("unsupported action %q: use %q or %q", repo.Action, types.GitActionClone, types.GitActionPull))
			track.item("", repo.Name, repo.Action, types.StatusFailed, "unsupported action", 0)
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
				recordFailure("git", repo.Name, fmt.Errorf("checking/updating remote URL: %w", err))
				track.item("", repo.Name, repo.Action, types.StatusFailed, err.Error(), time.Since(started))
				continue
			}

			// Pull latest changes
			err = helpers.HandleGitPull(gitOpts, initConfig)
			if err != nil {
				recordFailure("git", repo.Name, fmt.Errorf("pulling latest changes: %w", err))
				track.item("", repo.Name, repo.Action, types.StatusFailed, err.Error(), time.Since(started))
				continue
			}
			log.Infof("Git repository %s updated successfully", repo.Name)
			track.item("", repo.Name, repo.Action, types.StatusOK, "", time.Since(started))
		} else if os.IsNotExist(err) {
			if repo.Action == types.GitActionPull {
				recordFailure("git", repo.Name,
					fmt.Errorf("action is %q but nothing is checked out at %s", types.GitActionPull, gitOpts.Target))
				track.item("", repo.Name, repo.Action, types.StatusFailed, "nothing checked out at target", time.Since(started))
				continue
			}
			// Repository doesn't exist, clone it
			err = helpers.HandleGitClone(gitOpts, initConfig)
			if err != nil {
				recordFailure("git", repo.Name, fmt.Errorf("cloning: %w", err))
				track.item("", repo.Name, repo.Action, types.StatusFailed, err.Error(), time.Since(started))
				continue
			}
			log.Infof("Git repository %s cloned successfully", repo.Name)
			track.item("", repo.Name, repo.Action, types.StatusOK, "", time.Since(started))
		} else {
			// Some other error occurred
			recordFailure("git", repo.Name, fmt.Errorf("checking repository path: %w", err))
			track.item("", repo.Name, repo.Action, types.StatusFailed, err.Error(), time.Since(started))
			continue
		}
	}
	return nil
}

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
