package helpers

import (
	"encoding/base64"
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
// input. Nothing decoded it: base64 has no newlines and no "ssh-" prefix, so
// it fell through to the file-path branch and rwr tried to open the blob as a
// filename, failing with "file name too long". set_as_rwr_ssh_key produced a
// value that could never authenticate anything.
func TestSSHAuthAcceptsBase64EncodedKey(t *testing.T) {
	_, pem := generateKey(t)
	encoded := base64.StdEncoding.EncodeToString(pem)

	if err := sshAuthFor(encoded); err != nil {
		t.Fatalf("base64-encoded key rejected: %v", err)
	}
}

// The two forms that already worked must keep working.
func TestSSHAuthAcceptsRawPEMAndPath(t *testing.T) {
	path, pem := generateKey(t)

	if err := sshAuthFor(string(pem)); err != nil {
		t.Errorf("raw PEM rejected: %v", err)
	}
	if err := sshAuthFor(path); err != nil {
		t.Errorf("key path rejected: %v", err)
	}
}

// A path that happens to be valid base64 is still a path. "config" decodes
// cleanly under StdEncoding, so the decode alone cannot be the test - the
// decoded bytes have to look like a key.
func TestSSHAuthTreatsBase64LookingPathAsPath(t *testing.T) {
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
	junk := strings.Repeat("z", 400)

	err := sshAuthFor(junk)
	if err == nil {
		t.Fatal("expected an error for a value that is neither a key nor a path")
	}
	if !strings.Contains(err.Error(), "neither a readable file nor recognisable key material") {
		t.Errorf("unhelpful error: %v", err)
	}
	if strings.Contains(err.Error(), junk) {
		t.Error("the error repeats the whole value; it may be a private key")
	}
}
