package helpers

import (
	"errors"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestExplainGitPullFailure_NonFastForwardProtectsLocalWork(t *testing.T) {
	t.Parallel()

	err := explainGitPullFailure("/work/rwr", git.ErrNonFastForwardUpdate)
	if err == nil {
		t.Fatal("explainGitPullFailure returned nil")
	}
	if !errors.Is(err, git.ErrNonFastForwardUpdate) {
		t.Fatalf("error = %v, want ErrNonFastForwardUpdate", err)
	}
	var divergence *GitDivergenceError
	if !errors.As(err, &divergence) {
		t.Fatalf("error = %T, want *GitDivergenceError", err)
	}
	if divergence.Path != "/work/rwr" {
		t.Errorf("path = %q, want /work/rwr", divergence.Path)
	}
}

func TestExplainGitPullFailure_OtherErrorsUseExistingPath(t *testing.T) {
	t.Parallel()

	if got := explainGitPullFailure("/work/rwr", errors.New("network unavailable")); got != nil {
		t.Fatalf("error = %v, want nil", got)
	}
}
