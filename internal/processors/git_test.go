package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// The action field was decoded and never read, so `action: banana` was accepted
// and `action: pull` cloned anyway.
func TestProcessGitRepositories_RejectsUnknownAction(t *testing.T) {
	resetFailures()
	t.Cleanup(resetFailures)

	repos := []types.Git{{Name: "dotfiles", URL: "https://example.com/x.git", Path: t.TempDir(), Action: "banana"}}
	if err := processGitRepositories(repos, newTestInitConfig()); err != nil {
		t.Fatalf("processGitRepositories: %v", err)
	}

	err := failureError()
	if err == nil {
		t.Fatal("an unsupported git action was accepted silently")
	}
	if !strings.Contains(err.Error(), "banana") {
		t.Errorf("failure does not name the bad action: %v", err)
	}
}

// `action: pull` against a path with nothing checked out used to clone instead.
func TestProcessGitRepositories_PullDoesNotClone(t *testing.T) {
	resetFailures()
	t.Cleanup(resetFailures)

	target := filepath.Join(t.TempDir(), "missing")
	repos := []types.Git{{Name: "dotfiles", URL: "https://example.com/x.git", Path: target, Action: "pull"}}
	if err := processGitRepositories(repos, newTestInitConfig()); err != nil {
		t.Fatalf("processGitRepositories: %v", err)
	}

	if _, statErr := os.Stat(target); statErr == nil {
		t.Error("pull created the target; it must not clone")
	}
	if err := failureError(); err == nil || !strings.Contains(err.Error(), "pull") {
		t.Errorf("pull against a missing checkout was not reported: %v", err)
	}
}

func TestProcessGitRepositories_DivergenceIsProtectedSkip(t *testing.T) {
	resetFailures()
	t.Cleanup(resetFailures)

	root := t.TempDir()
	remotePath := filepath.Join(root, "remote.git")
	seedPath := filepath.Join(root, "seed")
	targetPath := filepath.Join(root, "target")
	otherPath := filepath.Join(root, "other")

	if _, err := git.PlainInit(remotePath, true); err != nil {
		t.Fatal(err)
	}
	seed, err := git.PlainInit(seedPath, false)
	if err != nil {
		t.Fatal(err)
	}
	commitFile(t, seed, seedPath, "base.txt", "base\n", "base")
	if _, err := seed.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{remotePath}}); err != nil {
		t.Fatal(err)
	}
	if err := seed.Push(&git.PushOptions{}); err != nil {
		t.Fatal(err)
	}

	target, err := git.PlainClone(targetPath, false, &git.CloneOptions{URL: remotePath})
	if err != nil {
		t.Fatal(err)
	}
	other, err := git.PlainClone(otherPath, false, &git.CloneOptions{URL: remotePath})
	if err != nil {
		t.Fatal(err)
	}
	localHead := commitFile(t, target, targetPath, "local.txt", "local work\n", "local")
	commitFile(t, other, otherPath, "remote.txt", "remote work\n", "remote")
	if err := other.Push(&git.PushOptions{}); err != nil {
		t.Fatal(err)
	}

	repos := []types.Git{{Name: "rwr", URL: remotePath, Path: targetPath, Action: "clone"}}
	if err := processGitRepositories(repos, newTestInitConfig()); err != nil {
		t.Fatalf("processGitRepositories: %v", err)
	}
	if err := failureError(); err != nil {
		t.Fatalf("diverged repository was recorded as a failure: %v", err)
	}
	after, err := target.Head()
	if err != nil {
		t.Fatal(err)
	}
	if after.Hash() != localHead {
		t.Fatalf("target HEAD changed from %s to %s", localHead, after.Hash())
	}
	if got, err := os.ReadFile(filepath.Join(targetPath, "local.txt")); err != nil || string(got) != "local work\n" {
		t.Fatalf("local work changed: content=%q err=%v", got, err)
	}
}

func commitFile(t *testing.T, repo *git.Repository, root, name, content, message string) plumbing.Hash {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worktree.Add(name); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit(message, &git.CommitOptions{Author: &object.Signature{Name: "RWR Test", Email: "rwr@example.invalid", When: time.Unix(1, 0)}})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
