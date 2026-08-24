package helpers

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

func TestCheckAndUpdateRemoteURLPreservesEquivalentSSHTransport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	const sshURL = "git@github.com:TheFynx/rwr-blueprints.git"
	if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{sshURL}}); err != nil {
		t.Fatal(err)
	}

	if err := CheckAndUpdateRemoteURL(dir, "https://github.com/thefynx/rwr-blueprints.git"); err != nil {
		t.Fatalf("CheckAndUpdateRemoteURL: %v", err)
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatal(err)
	}
	if got := remote.Config().URLs[0]; got != sshURL {
		t.Fatalf("origin URL = %q, want existing SSH URL %q", got, sshURL)
	}
}

func TestCheckAndUpdateRemoteURLChangesDifferentRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"git@github.com:TheFynx/old.git"}}); err != nil {
		t.Fatal(err)
	}
	const desired = "https://github.com/TheFynx/new.git"

	if err := CheckAndUpdateRemoteURL(dir, desired); err != nil {
		t.Fatalf("CheckAndUpdateRemoteURL: %v", err)
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatal(err)
	}
	if got := remote.Config().URLs[0]; got != desired {
		t.Fatalf("origin URL = %q, want %q", got, desired)
	}
}
