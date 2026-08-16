package prompts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/credentials"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// mapKeyring is the test stand-in for the OS keyring.
type mapKeyring struct {
	entries     map[string]string
	unavailable bool
	setErr      error
}

func (m *mapKeyring) Get(name string) (string, error) {
	if m.unavailable {
		return "", errors.New("no keyring backend")
	}
	value, ok := m.entries[name]
	if !ok {
		return "", credentials.ErrKeyringNotFound
	}
	return value, nil
}

func (m *mapKeyring) Set(name, value string) error {
	if m.unavailable {
		return credentials.ErrKeyringUnavailable
	}
	if m.setErr != nil {
		return m.setErr
	}
	m.entries[name] = value
	return nil
}

func TestSaveGitHubTokenDoesNotFallBackOnKeyringWriteFailure(t *testing.T) {
	configFile := withSaveFixture(t, &mapKeyring{setErr: errors.New("keyring is locked")})

	err := SaveGitHubTokenToConfig("gho_devicetoken", &types.InitConfig{})
	if err == nil {
		t.Fatal("SaveGitHubTokenToConfig = nil, want keyring error")
	}
	if got := viper.GetString("repository.gh_api_token"); got != "" {
		t.Fatalf("config token = %q, want empty", got)
	}
	if raw, readErr := os.ReadFile(configFile); readErr == nil && strings.Contains(string(raw), "gho_devicetoken") {
		t.Error("config file gained token after keyring write failure")
	}
}

func withSaveFixture(t *testing.T, ring credentials.Keyring) string {
	t.Helper()
	origRing := credentials.Ring
	credentials.Ring = ring
	viper.Reset()
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	viper.SetConfigFile(configFile)
	t.Cleanup(func() {
		credentials.Ring = origRing
		viper.Reset()
		types.RegisterCredentials(nil)
	})
	return configFile
}

// With a keyring available the device-flow token lands there and no plaintext
// file gains the token value - the spec's hard rule for managed credentials.
func TestSaveGitHubTokenPrefersKeyring(t *testing.T) {
	ring := &mapKeyring{entries: map[string]string{}}
	configFile := withSaveFixture(t, ring)

	initConfig := &types.InitConfig{}
	if err := SaveGitHubTokenToConfig("gho_devicetoken", initConfig); err != nil {
		t.Fatalf("SaveGitHubTokenToConfig: %v", err)
	}

	if ring.entries["gh_api_token"] != "gho_devicetoken" {
		t.Error("token did not reach the keyring")
	}
	if raw, err := os.ReadFile(configFile); err == nil && strings.Contains(string(raw), "gho_devicetoken") {
		t.Error("the config file gained the token despite an available keyring")
	}
	if value, _ := types.CredentialValue("gh_api_token"); value != "gho_devicetoken" {
		t.Error("the run's credential registry was not updated")
	}
}

// The grandfathered path: no keyring backend falls back to the config file -
// still restricted to 0600, and the file keeps being read on later runs.
func TestSaveGitHubTokenFallsBackToConfigFile(t *testing.T) {
	configFile := withSaveFixture(t, &mapKeyring{unavailable: true})

	initConfig := &types.InitConfig{}
	if err := SaveGitHubTokenToConfig("gho_devicetoken", initConfig); err != nil {
		t.Fatalf("SaveGitHubTokenToConfig: %v", err)
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("reading config file: %v", err)
	}
	if !strings.Contains(string(raw), "gho_devicetoken") {
		t.Error("fallback did not write the token to the config file")
	}
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config file mode = %o, want 0600", info.Mode().Perm())
	}
	if got := viper.GetString("repository.gh_api_token"); got != "gho_devicetoken" {
		t.Error("a config-file token must keep resolving (grandfathered read)")
	}
}
