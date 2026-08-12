package helpers

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// generateKey writes a real ed25519 keypair and returns its path and PEM.
// A hand-rolled fixture would not prove anything here: the point is that what
// ssh-keygen produces, and what rwr stores about it, round-trips through
// go-git.
func generateKey(t *testing.T) (path string, pem []byte) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	path = filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C", "rwr-test", "-f", path).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen failed: %v: %s", err, out)
	}
	pem, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading generated key: %v", err)
	}
	return path, pem
}

func sshAuthFor(value string) (err error) {
	cfg := &types.InitConfig{}
	cfg.Variables.Flags.SSHKey = value
	_, err = getSSHAuthMethod(cfg)
	return err
}

// The regression this file exists for.
//
// setAsRWRSSHKey stores base64 of the private key under
// repository.ssh_private_key, and --ssh-key documents base64 as an accepted
// input. Nothing decoded it: base64 has no newlines, so it fell through to the
// file-path branch and rwr tried to open the blob as a filename, failing with
// "file name too long". set_as_rwr_ssh_key produced a value that could never
// authenticate anything.
func TestSSHAuthAcceptsBase64EncodedKey(t *testing.T) {
	t.Parallel()

	_, pem := generateKey(t)
	encoded := base64.StdEncoding.EncodeToString(pem)

	if err := sshAuthFor(encoded); err != nil {
		t.Fatalf("base64-encoded key rejected: %v", err)
	}
}

// The two forms that already worked must keep working.
func TestSSHAuthAcceptsRawPEMAndPath(t *testing.T) {
	t.Parallel()

	path, pem := generateKey(t)

	if err := sshAuthFor(string(pem)); err != nil {
		t.Errorf("raw PEM rejected: %v", err)
	}
	if err := sshAuthFor(path); err != nil {
		t.Errorf("key path rejected: %v", err)
	}
}

// A key file whose name begins with "ssh-" is a path, not key material.
//
// The classifier used to shortcut on that prefix, so "ssh-key" and
// "ssh-private.pem" were parsed as key data and failed with "ssh: no key
// found" while the same file's absolute path worked. The prefix only ever
// matches a public key, which this flow cannot authenticate with, so the
// shortcut could not enable any working case - it could only swallow paths.
func TestSSHAuthTreatsSSHPrefixedNameAsPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, pem := generateKey(t)

	for _, name := range []string{"ssh-key", "ssh-private.pem", "ssh-rsa"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, pem, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := sshAuthFor(path); err != nil {
			t.Errorf("key file named %q rejected: %v", name, err)
		}
	}
}

// A public key is not a credential this flow can use. It must not be mistaken
// for one, whether supplied directly or base64-encoded.
func TestSSHAuthRejectsAPublicKey(t *testing.T) {
	t.Parallel()

	const pub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHhkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA user@host"

	if err := sshAuthFor(pub); err == nil {
		t.Error("a public key was accepted as an authentication credential")
	}
	if err := sshAuthFor(base64.StdEncoding.EncodeToString([]byte(pub))); err == nil {
		t.Error("a base64-encoded public key was accepted as an authentication credential")
	}
}

// A path that happens to be valid base64 is still a path. "config" decodes
// cleanly under StdEncoding, so the decode alone cannot be the test - the
// decoded bytes have to look like a key.
func TestSSHAuthTreatsBase64LookingPathAsPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// "config" is 6 chars, a multiple of 3, so it round-trips as base64.
	name := base64.StdEncoding.EncodeToString([]byte("config"))
	path := filepath.Join(dir, name)

	_, pem := generateKey(t)
	if err := os.WriteFile(path, pem, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := sshAuthFor(path); err != nil {
		t.Fatalf("base64-looking path rejected: %v", err)
	}
}

// A value that is neither says so, rather than reporting a stat error about a
// filename the operator never wrote. The message must not carry the whole
// value, which may be the key itself.
func TestSSHAuthRejectsUnusableValueClearly(t *testing.T) {
	t.Parallel()

	junk := strings.Repeat("z", 400)

	err := sshAuthFor(junk)
	if err == nil {
		t.Fatal("expected an error for a value that is neither a key nor a path")
	}
	// The sentinel, not the message text: the condition is part of the
	// contract, the wording is not.
	if !errors.Is(err, ErrUnusableSSHKey) {
		t.Errorf("error is not ErrUnusableSSHKey: %v", err)
	}
	if strings.Contains(err.Error(), junk) {
		t.Error("the error repeats the whole value; it may be a private key")
	}
}
