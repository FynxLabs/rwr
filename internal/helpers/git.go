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

	// Use authentication for private repos or SSH URLs
	if opts.Private || strings.HasPrefix(opts.URL, "git@") {
		auth = getAuthMethod(opts.URL, initConfig)
		log.Debugf("Using authentication for Git clone: %s", opts.URL)
	} else {
		log.Debugf("No authentication needed for Git clone: %s", opts.URL)
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

	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("error getting remote 'origin': %v", err)
	}

	currentURL := remote.Config().URLs[0]
	if currentURL != desiredURL {
		log.Infof("Updating remote URL from %s to %s", currentURL, desiredURL)

		// Remove the existing remote
		err = repo.DeleteRemote("origin")
		if err != nil {
			return fmt.Errorf("error removing existing remote: %v", err)
		}

		// Create a new remote with the updated URL
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{desiredURL},
		})
		if err != nil {
			return fmt.Errorf("error creating new remote with updated URL: %v", err)
		}
		log.Infof("Remote URL updated successfully")
	}

	return nil
}

func getAuthMethod(url string, initConfig *types.InitConfig) transport.AuthMethod {
	if strings.HasPrefix(url, "git@") {
		// Use the specified SSH key or try to find a suitable key
		sshKeyValue := initConfig.Variables.Flags.SSHKey

		// Check if the value is a Base64-encoded key or a file path
		if sshKeyValue != "" {
			// If the value contains newlines or begins with "ssh-", it's likely a Base64-encoded key
			if strings.Contains(sshKeyValue, "\n") || strings.HasPrefix(sshKeyValue, "ssh-") {
				log.Debugf("Using Base64-encoded SSH key")
				auth, err := ssh.NewPublicKeys("git", []byte(sshKeyValue), "")
				if err != nil {
					log.Errorf("Error creating SSH authentication from Base64 key: %v", err)
					return nil
				}
				return auth
			} else {
				// Treat as a file path
				if _, err := os.Stat(sshKeyValue); os.IsNotExist(err) {
					log.Errorf("Specified SSH key file does not exist: %s", sshKeyValue)
					return nil
				}

				auth, err := ssh.NewPublicKeysFromFile("git", sshKeyValue, "")
				if err != nil {
					log.Errorf("Error creating SSH authentication from file: %v", err)
					return nil
				}
				return auth
			}
		} else {
			// No SSH key specified, try to find a suitable key file
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Errorf("Error getting user home directory: %v", err)
				return nil
			}

			// List of common SSH key locations to try
			possibleKeys := []string{
				filepath.Join(homeDir, ".ssh", "id_rsa"),
				filepath.Join(homeDir, ".ssh", "git"),
				filepath.Join(homeDir, ".ssh", "id_ed25519"),
				filepath.Join(homeDir, ".ssh", "github_rsa"),
				filepath.Join(homeDir, ".ssh", "id_ecdsa"),
			}

			// Try each possible key location
			keyFound := false
			sshKeyPath := ""
			for _, keyPath := range possibleKeys {
				if _, err := os.Stat(keyPath); err == nil {
					sshKeyPath = keyPath
					keyFound = true
					log.Debugf("Found SSH key at: %s", sshKeyPath)
					break
				}
			}

			if !keyFound {
				log.Errorf("No SSH key found in common locations. Please specify an SSH key path or Base64-encoded key.")
				return nil
			}

			auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
			if err != nil {
				log.Errorf("Error creating SSH authentication: %v", err)
				return nil
			}
			return auth
		}
	} else {
		return &http.BasicAuth{
			Username: "git",
			Password: initConfig.Variables.Flags.GHAPIToken,
		}
	}
}

func HandleGitPull(opts types.GitOptions, initConfig *types.InitConfig) error {
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

	// Get the remote URL to determine if we need authentication
	remote, err := repo.Remote("origin")
	if err != nil {
		log.Errorf("Error getting remote 'origin': %v", err)
		return fmt.Errorf("error getting remote 'origin': %v", err)
	}

	var auth transport.AuthMethod
	if len(remote.Config().URLs) > 0 {
		remoteURL := remote.Config().URLs[0]
		// For SSH URLs (git@...) or if the repo is marked as private, use authentication
		if strings.HasPrefix(remoteURL, "git@") || opts.Private {
			auth = getAuthMethod(remoteURL, initConfig)
			log.Debugf("Using authentication for Git pull: %s", opts.Target)
		} else {
			log.Debugf("No authentication needed for Git pull: %s", opts.Target)
		}
	}

	err = worktree.Pull(&git.PullOptions{
		Auth: auth,
	})
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
