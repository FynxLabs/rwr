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
	moduleCache := dag.CacheVolume("go-modules")
	buildCache := dag.CacheVolume("go-build")

	src := dag.Directory()

	// Configure Go environment
	goEnv := dag.Go().
		WithModuleCache(moduleCache).
		WithBuildCache(buildCache).
		WithSource(src).
		WithCgoDisabled()

	// Run tests
	testResult, err := goEnv.
		Exec([]string{"test", "-v", "./..."}).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("test failed: %w", err)
	}

	// If this is a tag (v*), also do release
	if strings.HasPrefix(ref, "v") {
		releaseContainer := dag.Goreleaser().Base()

		// Add secrets
		releaseContainer = releaseContainer.
			WithSecretVariable("GITHUB_TOKEN", githubToken).
			WithSecretVariable("HOMEBREW_TAP_DEPLOY_KEY", homebrewToken)

		// Run release
		releaseResult, err := releaseContainer.
			WithWorkdir("/src").
			WithMountedDirectory("/src", src).
			WithExec([]string{
				"goreleaser",
				"release",
				"--clean",
			}).
			Stdout(ctx)
		if err != nil {
			return "", fmt.Errorf("release failed: %w", err)
		}
		return fmt.Sprintf("Tests passed:\n%s\nRelease completed:\n%s", testResult, releaseResult), nil
	}

	return fmt.Sprintf("Tests passed:\n%s", testResult), nil
}
