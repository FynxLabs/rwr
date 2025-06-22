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
	githubToken *dagger.Secret,
	homebrewToken *dagger.Secret,
) (string, error) {
	src := dag.Directory()

	// Configure Go environment
	goEnv := dag.Go().
		WithSource(src).
		WithCgoDisabled().
		WithEnvVariable("GO111MODULE", "on").
		WithExec([]string{"go", "mod", "tidy"})

	// Run tests
	testResult := goEnv.
		WithExec([]string{"go", "test", "-v", "./..."})

	// Only do release for tags
	if strings.HasPrefix(ref, "v") {
		args := []string{"release", "--clean"}

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
		return fmt.Sprintf("Tests passed:\n%v\nRelease completed:\n%v", testResult, releaseResult), nil
	}

	return fmt.Sprintf("Tests passed:\n%v", testResult), nil
}
