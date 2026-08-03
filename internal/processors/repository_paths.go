// Path containment: every path a repository step reads, writes, or
// removes resolves through here, and anything outside the provider's
// declared boundaries is refused. Steps run with the provider's
// privileges — root, for system package managers.

package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// isLocalPath reports whether a repository URL names a file on this machine
// rather than something to fetch over the network.
func isLocalPath(url string) bool {
	if url == "" {
		return false
	}
	if strings.HasPrefix(url, "file://") {
		return true
	}
	scheme, _, found := strings.Cut(url, "://")
	return !found || scheme == ""
}

// localPath is the filesystem path a local repository URL names.
func localPath(url string) string {
	if !isLocalPath(url) {
		return ""
	}
	return system.ExpandPath(strings.TrimPrefix(url, "file://"))
}

// repositoryFilePath resolves the file an in-place edit step names and refuses
// any file the provider does not declare in [provider.repository.paths].
//
// It is the containment removeRepositoryPath applies, widened by one case: a
// declared path may itself be the file being edited, because /etc/pacman.conf
// and macports' sources.conf are files rather than directories.
func repositoryFilePath(path string, paths types.RepositoryPaths) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("step has no path")
	}

	resolved := filepath.Clean(system.ExpandPath(path))
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("refusing to edit %q: path is not absolute", path)
	}

	boundaries := repositoryBoundaries(paths)
	if len(boundaries) == 0 {
		return "", fmt.Errorf("refusing to edit %q: provider declares no repository paths", resolved)
	}

	for _, boundary := range boundaries {
		if resolved == boundary {
			return resolved, nil
		}
	}
	if withinAny(resolved, boundaries) {
		return resolved, nil
	}

	return "", fmt.Errorf("refusing to edit %q: outside the provider's repository paths %v", resolved, boundaries)
}

// removeRepositoryPath deletes one path named by a provider's "remove" step.
//
// The path is a provider template rendered against blueprint values, and the
// deletion runs with whatever privilege the provider declares — root, for every
// system package manager. So it is only carried out inside the directories the
// provider itself declares in [provider.repository.paths]: neither a provider
// definition nor a repository name may redirect a root-owned delete at
// /etc/passwd. Symlinked directory components are resolved before the check, so a
// planted link cannot lead the delete out of those directories either.
func removeRepositoryPath(path string, paths types.RepositoryPaths, elevated, debug bool) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("remove step has no path")
	}

	resolved := filepath.Clean(system.ExpandPath(path))
	if !filepath.IsAbs(resolved) {
		return fmt.Errorf("refusing to remove %q: path is not absolute", path)
	}

	boundaries := repositoryBoundaries(paths)
	if len(boundaries) == 0 {
		return fmt.Errorf("refusing to remove %q: provider declares no repository paths", resolved)
	}

	if !withinAny(resolved, boundaries) {
		return fmt.Errorf("refusing to remove %q: outside the provider's repository paths %v", resolved, boundaries)
	}

	realPath, err := realRemovalPath(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			// `rwr all` is expected to be run repeatedly, and a removal that has
			// already happened is the state the blueprint asked for.
			log.Debugf("Repository path already absent: %s", resolved)
			return nil
		}
		return fmt.Errorf("error resolving %q: %w", resolved, err)
	}

	if !withinAny(realPath, boundaries) {
		return fmt.Errorf("refusing to remove %q: %q is outside the provider's repository paths %v", resolved, realPath, boundaries)
	}

	if system.IsDryRun() {
		log.Infof("[DRY-RUN] Would remove file: %s", resolved)
		return nil
	}

	if elevated {
		// Sources lists and keyrings are root-owned, so the delete needs the same
		// elevation the writes that created them used. "-f" keeps it idempotent.
		return system.RunCommand(types.Command{
			Exec:     "rm",
			Args:     []string{"-f", "--", resolved},
			Elevated: true,
		}, debug)
	}

	if err := os.Remove(resolved); err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Repository path already absent: %s", resolved)
			return nil
		}
		return err
	}

	log.Infof("Removed repository file: %s", resolved)
	return nil
}

// repositoryTempDir is the per-run private directory repository steps stage
// into ({{ .TempKeyPath }}). It replaces fixed paths under os.TempDir(): a
// world-known name like /tmp/docker.gpg can be pre-created (or swapped
// between steps) by any local user, and the very next step imports it as a
// signing key with root privileges. A 0700 per-run directory is not
// reachable by other users at all.
var (
	repoTempDirOnce sync.Once
	repoTempDirPath string
	repoTempDirErr  error
)

// repositoryTempDir returns the run's private staging directory, or the error
// that prevented creating it. A failure used to fall back to the predictable
// <tmp>/rwr-repo-unavailable — a fixed, world-known name any local user can
// pre-create, which is exactly the class {{ .TempDir }} exists to eliminate.
func repositoryTempDir() (string, error) {
	repoTempDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "rwr-repo-")
		if err != nil {
			repoTempDirErr = fmt.Errorf("could not create the repository staging directory: %w", err)
			return
		}
		repoTempDirPath = dir
	})
	return repoTempDirPath, repoTempDirErr
}

// repositoryWritePath resolves the destination a download/write/copy step
// names and refuses anything outside the provider's declared repository paths
// or the run's private staging directory. These steps run with the provider's
// privileges — root, for every system package manager — and the destination
// is a template rendered against blueprint values, so an unchecked dest is a
// root-privileged write to an arbitrary path. The in-place edit and remove
// steps have been contained since #163; this closes the same door for the
// steps that create files.
func repositoryWritePath(dest string, paths types.RepositoryPaths) (string, error) {
	if strings.TrimSpace(dest) == "" {
		return "", fmt.Errorf("step has no dest")
	}

	resolved := filepath.Clean(system.ExpandPath(dest))
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("refusing to write %q: path is not absolute", dest)
	}

	tempDir, err := repositoryTempDir()
	if err != nil {
		return "", err
	}
	boundaries := append(repositoryBoundaries(paths), tempDir)
	if resolved == tempDir {
		return "", fmt.Errorf("refusing to write %q: it is the staging directory itself", dest)
	}
	for _, boundary := range boundaries {
		if resolved == boundary {
			return resolved, nil
		}
	}
	if withinAny(resolved, boundaries) {
		return resolved, nil
	}

	return "", fmt.Errorf("refusing to write %q: outside the provider's repository paths %v", resolved, boundaries)
}

// repositoryBoundaries returns the directories a remove step may act inside.
func repositoryBoundaries(paths types.RepositoryPaths) []string {
	var boundaries []string
	for _, declared := range []string{paths.Sources, paths.Keys, paths.Config} {
		if declared == "" {
			continue
		}
		expanded := filepath.Clean(system.ExpandPath(declared))
		if !filepath.IsAbs(expanded) {
			continue
		}
		if evaluated, err := filepath.EvalSymlinks(expanded); err == nil {
			boundaries = append(boundaries, evaluated)
		}
		boundaries = append(boundaries, expanded)
	}
	return boundaries
}

// realRemovalPath resolves symlinked directory components of path, leaving the
// final element alone: removing a symlink is meant to remove the link itself.
func realRemovalPath(path string) (string, error) {
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

// withinAny reports whether path sits strictly under one of the boundaries. The
// boundary directory itself is not removable: no provider asks for it, and
// deleting /etc/apt/sources.list.d wholesale is never what a blueprint meant.
func withinAny(path string, boundaries []string) bool {
	for _, boundary := range boundaries {
		if strings.HasPrefix(path, boundary+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
