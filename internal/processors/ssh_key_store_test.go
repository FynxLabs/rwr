package processors

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/credentials"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sshStoreKeyring struct {
	entries     map[string]string
	unavailable bool
}

func (r *sshStoreKeyring) Get(name string) (string, error) {
	if r.unavailable {
		return "", errors.New("no keyring backend")
	}
	value, ok := r.entries[name]
	if !ok {
		return "", credentials.ErrKeyringNotFound
	}
	return value, nil
}

func (r *sshStoreKeyring) Set(name, value string) error {
	if r.unavailable {
		return errors.New("no keyring backend")
	}
	if r.entries == nil {
		r.entries = map[string]string{}
	}
	r.entries[name] = value
	return nil
}

func withSSHStoreKeyring(t *testing.T, ring credentials.Keyring) {
	t.Helper()
	previous := credentials.Ring
	credentials.Ring = ring
	t.Cleanup(func() {
		credentials.Ring = previous
		types.RegisterCredentials(nil)
	})
}

// withConfigFile points viper at a throwaway config and returns its path.
func withConfigFile(t *testing.T) string {
	t.Helper()
	viper.Reset()
	path := filepath.Join(t.TempDir(), "config.yaml")
	viper.SetConfigFile(path)
	t.Cleanup(viper.Reset)
	return path
}

// writeTestKey generates a real private key and returns its path.
func writeTestKey(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C", "rwr-test", "-f", path).CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen failed: %v: %s", err, out)
	}
	return path
}

// The config file holds a private key when this is done, so it must never be
// left readable by anyone else. viper writes at 0644 with no way to ask for
// narrower, so the mode has to be established before the write rather than
// repaired after it.
func TestSetAsRWRSSHKeyLeavesConfigOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	configPath := withConfigFile(t)
	withSSHStoreKeyring(t, &sshStoreKeyring{unavailable: true})
	keyPath := writeTestKey(t)

	require.NoError(t, setAsRWRSSHKey(keyPath))

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, helpers.ConfigFilePerm, info.Mode().Perm(),
		"config file holding a private key is not owner-only")
}

// The realistic starting state: a config file left at 0644 by an older rwr, or
// by an operator's editor. It must be narrowed before the key lands in it, not
// after.
func TestSetAsRWRSSHKeyNarrowsAnExistingLooseConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	configPath := withConfigFile(t)
	withSSHStoreKeyring(t, &sshStoreKeyring{unavailable: true})
	require.NoError(t, os.WriteFile(configPath, []byte("repository: {}\n"), 0o644))
	keyPath := writeTestKey(t)

	require.NoError(t, setAsRWRSSHKey(keyPath))

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, helpers.ConfigFilePerm, info.Mode().Perm())
}

// What gets stored has to be what the git auth path can read back. It stores
// base64 of the PEM; for a long time nothing decoded that, so this recorded a
// value no clone could ever use.
func TestSetAsRWRSSHKeyStoresAValueGitAuthCanUse(t *testing.T) {
	withConfigFile(t)
	withSSHStoreKeyring(t, &sshStoreKeyring{unavailable: true})
	keyPath := writeTestKey(t)

	require.NoError(t, setAsRWRSSHKey(keyPath))

	stored := viper.GetString("repository.ssh_private_key")
	require.NotEmpty(t, stored, "no key recorded")

	decoded, err := base64.StdEncoding.DecodeString(stored)
	require.NoError(t, err, "stored value is not the base64 form the reader expects")

	original, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, original, decoded, "stored key does not round-trip to the file it came from")
}

func TestSetAsRWRSSHKeyPrefersKeyring(t *testing.T) {
	configPath := withConfigFile(t)
	ring := &sshStoreKeyring{entries: map[string]string{}}
	withSSHStoreKeyring(t, ring)
	keyPath := writeTestKey(t)

	require.NoError(t, setAsRWRSSHKey(keyPath))

	stored := ring.entries["ssh_private_key"]
	require.NotEmpty(t, stored, "private key did not reach the keyring")
	if raw, err := os.ReadFile(configPath); err == nil && len(raw) != 0 {
		assert.NotContains(t, string(raw), stored, "config file gained the private key despite an available keyring")
	}
	value, ok := types.CredentialValue("ssh_private_key")
	require.True(t, ok)
	assert.Equal(t, stored, value)
}

func TestSetAsRWRSSHKeyClearsLegacyConfigAfterKeyringSave(t *testing.T) {
	configPath := withConfigFile(t)
	viper.Set("repository.ssh_private_key", "legacy-plaintext-key")
	require.NoError(t, viper.WriteConfig())

	ring := &sshStoreKeyring{entries: map[string]string{}}
	withSSHStoreKeyring(t, ring)
	keyPath := writeTestKey(t)
	require.NoError(t, setAsRWRSSHKey(keyPath))

	raw, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "legacy-plaintext-key")
	assert.NotContains(t, string(raw), ring.entries["ssh_private_key"])
	assert.Empty(t, viper.GetString("repository.ssh_private_key"))
}
