package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// A blueprint that writes `path: ~/.ssh` must still put the key in the home
// directory. Commands no longer pass through a shell, so nothing expands the ~
// on the way to ssh-keygen - without expansion the key lands in a directory
// literally named "~" under the working directory.
func TestSSHKeyPathExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}

	rec := exectest.New()
	defer system.SetExecutor(rec)()

	blueprint := []byte(`
ssh_keys:
  - name: id_test_rwr
    action: create
    type: ed25519
    path: ~/.ssh
    no_passphrase: true
    copy_to_github: false
`)

	// Ignore the error: this asserts on the command that was built, and the key
	// may already exist on the machine running the test.
	_ = ProcessSSHKeys(blueprint, t.TempDir(), "yaml", &types.OSInfo{}, &types.InitConfig{})

	calls := rec.Find("ssh-keygen")
	if len(calls) == 0 {
		t.Skip("ssh-keygen was not invoked (key may already exist)")
	}

	args := strings.Join(calls[0].Args, " ")
	if strings.Contains(args, "~") {
		t.Errorf("ssh-keygen received an unexpanded ~: %v", calls[0].Args)
	}
	want := filepath.Join(home, ".ssh", "id_test_rwr")
	if !strings.Contains(args, want) {
		t.Errorf("ssh-keygen args = %v, want them to contain %q", calls[0].Args, want)
	}
}
