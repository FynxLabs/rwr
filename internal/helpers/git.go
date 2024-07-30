package helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func HandleGitOperation(opts types.GitOptions, initConfig *types.InitConfig) error {
	if filepath.Ext(opts.URL) != "" {
		return HandleGitFileDownload(opts, initConfig)
	} else {
		return HandleGitClone(opts, initConfig)
	}
}

func HandleGitClone(opts types.GitOptions, initConfig *types.InitConfig) error {
	var auth transport.AuthMethod

	log.Debugf("Cloning Git repository: %s", opts.URL)

	if opts.Private {
		auth = getAuthMethod(opts.URL, initConfig)
	}

	targetDir := filepath.Dir(opts.Target)
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	_, err = git.PlainClone(opts.Target, false, &git.CloneOptions{
		URL:  opts.URL,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("error cloning Git repository: %v", err)
	}

	log.Infof("Git repository cloned to: %s", opts.Target)

	// Check and update remote URL if necessary
	err = CheckAndUpdateRemoteURL(opts.Target, opts.URL)
	if err != nil {
		return fmt.Errorf("error checking/updating remote URL: %v", err)
	}

	return nil
}

func CheckAndUpdateRemoteURL(repoPath, desiredURL string) error {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("error opening Git repository: %v", err)
	}

	remoteConfig, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("error getting remote 'origin': %v", err)
	}

	currentURL := remoteConfig.Config().URLs[0]
	if currentURL != desiredURL {
		log.Infof("Updating remote URL from %s to %s", currentURL, desiredURL)
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{desiredURL},
		})
		if err != nil {
			return fmt.Errorf("error updating remote URL: %v", err)
		}
		log.Infof("Remote URL updated successfully")
	}

	return nil
}

func getAuthMethod(url string, initConfig *types.InitConfig) transport.AuthMethod {
	if strings.HasPrefix(url, "git@") {
		auth, err := ssh.NewPublicKeysFromFile("git", initConfig.Variables.Flags.SSHKey, "")
		if err != nil {
			log.Errorf("Error creating SSH authentication: %v", err)
			return nil
		}
		return auth
	} else {
		return &http.BasicAuth{
			Username: "git",
			Password: initConfig.Variables.Flags.GHAPIToken,
		}
	}
}

func HandleGitPull(opts types.GitOptions) error {
	log.Debugf("Pulling changes from Git repository: %s", opts.Target)
	repo, err := git.PlainOpen(opts.Target)
	if err != nil {
		log.Errorf("Error opening Git repository: %v", err)
		return fmt.Errorf("error opening Git repository: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		log.Errorf("Error getting worktree: %v", err)
		return fmt.Errorf("error getting worktree: %v", err)
	}

	err = worktree.Pull(&git.PullOptions{})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.Errorf("Error pulling changes from Git repository: %v", err)
		return fmt.Errorf("error pulling changes from Git repository: %v", err)
	}

	log.Infof("Git repository updated: %s", opts.Target)
	return nil
}

func HandleGitFileDownload(opts types.GitOptions, initConfig *types.InitConfig) error {
	log.Debugf("Downloading file from Git repository: %s", opts.URL)
	// Extract the repository URL and file path from the opts.URL
	parts := strings.Split(opts.URL, "/blob/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid URL format for file download")
	}
	repoURL := parts[0]
	filePath := parts[1]

	// Create a temporary directory for cloning the repository
	tempDir, err := os.MkdirTemp("", "git-clone-")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Errorf("error removing temporary directory: %v", err)
		}
	}(tempDir)

	// Clone the repository into the temporary directory
	cloneOpts := types.GitOptions{
		URL:     repoURL,
		Private: opts.Private,
		Target:  tempDir,
	}
	err = HandleGitClone(cloneOpts, initConfig)
	if err != nil {
		return fmt.Errorf("error cloning Git repository: %v", err)
	}

	// Read the contents of the specified file
	fileContent, err := os.ReadFile(filepath.Join(tempDir, filePath))
	if err != nil {
		return fmt.Errorf("error reading file from Git repository: %v", err)
	}

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(opts.Target)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	// Write the file contents to the target file
	err = os.WriteFile(opts.Target, fileContent, 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %v", err)
	}

	log.Infof("File downloaded from Git repository: %s", opts.Target)
	return nil
}
