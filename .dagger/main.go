package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/rwr/internal/dagger"
)

type Rwr struct{}

// CI handles both testing and releasing based on the context
func (m *Rwr) CI(
	ctx context.Context,
	ref string,
	skipPublish bool,
	githubToken *dagger.Secret,
	homebrewToken *dagger.Secret,
) (string, error) {
	src := dag.Directory()

	// Configure Go environment
	goEnv := dag.Go().
		WithSource(src).
		WithCgoDisabled()

	// Run tests
	testResult, err := goEnv.
		Exec([]string{"test", "-v", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("test failed: %w", err)
	}

	// If this is a tag (v*) or skipPublish is true, do release
	if strings.HasPrefix(ref, "v") || skipPublish {
		args := []string{"release", "--clean"}
		if skipPublish {
			args = append(args, "--skip=publish")
		}

		releaseContainer := dag.Goreleaser().Base().
			WithSecretVariable("GITHUB_TOKEN", githubToken).
			WithSecretVariable("HOMEBREW_TAP_DEPLOY_KEY", homebrewToken).
			WithWorkdir("/src").
			WithMountedDirectory("/src", src).
			WithExec(args)

		releaseResult, err := releaseContainer.Stdout(ctx)
		if err != nil {
			return "", fmt.Errorf("release failed: %w", err)
		}
		return fmt.Sprintf("Tests passed:\n%s\nRelease completed:\n%s", testResult, releaseResult), nil
	}

	return fmt.Sprintf("Tests passed:\n%s", testResult), nil
}
