package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
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
