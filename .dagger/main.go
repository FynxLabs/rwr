package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/rwr/internal/dagger"
)

type Rwr struct {
	// +private
	Source *dagger.Directory
}

func New(
	// The source directory for building RWR
	// +optional
	// +defaultPath="/"
	source *dagger.Directory,
) *Rwr {
	if source == nil {
		source = dag.Directory()
	}
	return &Rwr{Source: source}
}

// CI handles both testing and releasing based on the context
func (m *Rwr) CI(
	ctx context.Context,
	// Git reference (tag or branch)
	ref string,
	// GitHub token for releases
	githubToken *dagger.Secret,
	// Homebrew token for tap updates
	homebrewToken *dagger.Secret,
) (string, error) {
	// Run tests first
	testOutput, err := m.Test(ctx)
	if err != nil {
		return "", fmt.Errorf("tests failed: %w", err)
	}

	// Only do release for tags
	if strings.HasPrefix(ref, "v") {
		releaseOutput, err := m.Release(ctx, githubToken, homebrewToken)
		if err != nil {
			return "", fmt.Errorf("release failed: %w", err)
		}
		return fmt.Sprintf("Tests passed:\n%s\n\nRelease completed:\n%s", testOutput, releaseOutput), nil
	}

	return fmt.Sprintf("Tests passed:\n%s", testOutput), nil
}

// Test runs the test suite
func (m *Rwr) Test(ctx context.Context) (string, error) {
	return dag.Go().
		WithSource(m.Source).
		WithCgoDisabled().
		WithEnvVariable("GO111MODULE", "on").
		WithExec([]string{"go", "mod", "tidy"}).
		Container().
		WithExec([]string{"go", "test", "-v", "./..."}).
		Stdout(ctx)
}

// Release performs a release using goreleaser
func (m *Rwr) Release(
	ctx context.Context,
	// GitHub token for releases
	githubToken *dagger.Secret,
	// Homebrew token for tap updates
	homebrewToken *dagger.Secret,
) (string, error) {
	// Use the act3-ai goreleaser module with proper API
	return dag.Goreleaser(m.Source).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithSecretVariable("HOMEBREW_TAP_DEPLOY_KEY", homebrewToken).
		Release().
		WithClean().
		Run(ctx)
}

// Build compiles the RWR binary
func (m *Rwr) Build(
	ctx context.Context,
	// Target OS
	// +default="linux"
	os string,
	// Target architecture
	// +default="amd64"
	arch string,
) *dagger.File {
	return dag.Go().
		WithSource(m.Source).
		WithCgoDisabled().
		WithEnvVariable("GOOS", os).
		WithEnvVariable("GOARCH", arch).
		WithExec([]string{"go", "build", "-o", "rwr"}).
		Container().
		File("/work/src/rwr")
}

// Lint runs linting on the codebase
func (m *Rwr) Lint(ctx context.Context) (string, error) {
	return dag.Go().
		WithSource(m.Source).
		WithExec([]string{"go", "vet", "./..."}).
		WithExec([]string{"gofmt", "-l", "."}).
		Container().
		Stdout(ctx)
}

// Version returns the current version from git tags
func (m *Rwr) Version(ctx context.Context) (string, error) {
	return dag.Container().
		From("alpine/git:latest").
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithExec([]string{"git", "describe", "--tags", "--always"}).
		Stdout(ctx)
}
