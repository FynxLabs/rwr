package helpers

import (
	"errors"
	"fmt"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/viper"
)

func HandleGitOperation(opts types.GitOptions) error {
	// Determine the Git operation based on the URL
	if filepath.Ext(opts.URL) != "" {
		// Individual file download/read
		return HandleGitFileDownload(opts)
	} else {
		// Clone the Git repository
		return HandleGitClone(opts)
	}
}

func HandleGitClone(opts types.GitOptions) error {
	var auth transport.AuthMethod

	log.Debugf("Cloning Git repository: %s", opts.URL)

	if opts.Private {
		// Determine the authentication method based on the URL scheme and private flag
		if opts.URL[0:3] == "git" {
			// SSH authentication for private repositories
			privateKey := viper.GetString("repository.ssh_private_key")
			var err error
			auth, err = ssh.NewPublicKeysFromFile("git", privateKey, "")
			if err != nil {
				return fmt.Errorf("error creating SSH authentication: %v", err)
			}
		} else {
			// HTTPS authentication for private repositories
			token := viper.GetString("repository.gh_api_token")
			auth = &http.BasicAuth{
				Username: "git",
				Password: token,
			}
		}
	}

	// Create the target directory if it doesn't exist
	targetDir := filepath.Dir(opts.Target)
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating target directory: %v", err)
	}

	// Clone the Git repository
	_, err = git.PlainClone(opts.Target, false, &git.CloneOptions{
		URL:  opts.URL,
		Auth: auth,
		//Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", opts.Branch)),
	})
	if err != nil {
		return fmt.Errorf("error cloning Git repository: %v", err)
	}

	log.Infof("Git repository cloned to: %s", opts.Target)
	return nil
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

func HandleGitFileDownload(opts types.GitOptions) error {
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
	err = HandleGitClone(cloneOpts)
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
