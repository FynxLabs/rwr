package processors

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestGenerateSSHKeyCreatesDirectoryAndPreservesExistingKey(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen fixture unavailable")
	}
	key := types.SSHKey{Name: "git", Path: filepath.Join(t.TempDir(), "new-home", ".ssh"), Type: "ed25519", NoPassphrase: true}
	config := &types.InitConfig{}
	path, err := generateSSHKey(key, config)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".pub"); err != nil {
		t.Fatal("public key missing")
	}
	if _, err := generateSSHKey(key, config); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("existing private key replaced")
	}
}
