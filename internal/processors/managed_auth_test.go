package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/helpers"
	"github.com/freehold-digital/rwr/internal/types"
	"github.com/spf13/viper"
)

// writeInitFile writes an init file for LoadConfiguration to load.
func writeInitFile(t *testing.T, content string) string {
	t.Helper()
	initFile := filepath.Join(t.TempDir(), "init.yaml")
	if err := os.WriteFile(initFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing init file: %v", err)
	}
	return initFile
}

// resetManagedAuthState isolates the global registries LoadConfiguration mutates.
func resetManagedAuthState(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		types.RegisterCredentials(nil)
		types.SetExposedCredentials(nil)
		_ = os.Unsetenv("RWR_CRED_CACHIX_TOKEN")
	})
}

const declaredCredentialInit = `
blueprints:
  format: yaml
  location: "."

credentials:
  - name: cachix_token
    description: "Cachix auth token"
    sources: [env:CACHIX_AUTH_TOKEN]
`

func TestLoadConfigurationContinuesWithoutBitwarden(t *testing.T) {
	resetManagedAuthState(t)
	// No tools on PATH and no previously managed installation: this models
	// a fresh machine, rather than mocking the CLI result.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("RWR_CRED_GPG_PASSPHRASE", "")
	initFile := writeInitFile(t, `
blueprints:
  format: yaml
  location: "."
credentials:
  - name: gpg_passphrase
    sources: [bw:gpg-signing/password, prompt]
    scope: [scripts]
exposeCredentials: [gpg_passphrase]
`)
	config, err := LoadConfiguration(initFile, types.Flags{Interactive: false})
	if err != nil {
		t.Fatalf("missing optional Bitwarden stopped initialization: %v", err)
	}
	if config == nil {
		t.Fatal("no initialized configuration")
	}
	if _, ok := types.CredentialValue("gpg_passphrase"); ok {
		t.Fatal("skipping registered an empty secret")
	}
	if _, ok := types.ExportedCredentialEnv()["RWR_CRED_GPG_PASSPHRASE"]; ok {
		t.Fatal("skipping exported an empty secret")
	}
}

// A declared credential resolves at init time and stays withheld: absent from
// template scope and the RWR_CRED_* export until exposeCredentials names it -
// the same treatment the two built-ins get.
func TestLoadConfigurationResolvesAndWithholdsDeclaredCredential(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "cachix-secret-value")

	initFile := writeInitFile(t, declaredCredentialInit)
	config, err := LoadConfiguration(initFile, types.Flags{})
	if err != nil {
		t.Fatalf("LoadConfiguration: %v", err)
	}
	if len(config.Credentials) != 1 || config.Credentials[0].Name != "cachix_token" {
		t.Fatalf("credentials section decoded as %+v", config.Credentials)
	}
	if value, _ := types.CredentialValue("cachix_token"); value != "" {
		t.Errorf("initialization acquired credential %q", value)
	}

	// Withheld from the spawned-command env: no opt-in, no export.
	if os.Getenv("RWR_CRED_CACHIX_TOKEN") != "" {
		t.Error("RWR_CRED_CACHIX_TOKEN was exported without exposeCredentials")
	}

	// Withheld from template scope: rendering may error (missing key) but must
	// never produce the value.
	rendered, err := helpers.ResolveTemplate([]byte("v={{ .Credentials.cachix_token }}"), config.Variables)
	if err == nil && strings.Contains(string(rendered), "cachix-secret-value") {
		t.Errorf("template rendered an unexposed credential: %s", rendered)
	}
	if err != nil && strings.Contains(err.Error(), "cachix-secret-value") {
		t.Errorf("template error leaked the credential: %v", err)
	}
}

// With the exposeCredentials opt-in, the same credential reaches both surfaces.
func TestLoadConfigurationExposureDoesNotAcquire(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "cachix-secret-value")
	initFile := writeInitFile(t, declaredCredentialInit+"\nexposeCredentials: [cachix_token]\n")
	_, err := LoadConfiguration(initFile, types.Flags{})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := types.CredentialValue("cachix_token"); ok {
		t.Fatalf("initialization resolved %q", value)
	}
	if os.Getenv("RWR_CRED_CACHIX_TOKEN") != "" {
		t.Fatal("credential exported to parent process")
	}
}

func TestLoadConfigurationStrictDecodesCredentials(t *testing.T) {
	resetManagedAuthState(t)

	initFile := writeInitFile(t, `
blueprints:
  format: yaml
  location: "."

credentials:
  - name: cachix_token
    soruces: [keyring]
`)
	_, err := LoadConfiguration(initFile, types.Flags{})
	if err == nil || !strings.Contains(err.Error(), "soruces") {
		t.Fatalf("LoadConfiguration = %v, want a strict-decode error naming the unknown key", err)
	}
}

// Loading configuration never requires a declared credential to be available.
func TestLoadConfigurationUnavailableCredentialDoesNotFail(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "")
	if _, err := LoadConfiguration(writeInitFile(t, declaredCredentialInit), types.Flags{Interactive: true}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfigurationDoesNotResolveScopedCredential(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "")

	initFile := writeInitFile(t, `
blueprints:
  format: yaml
  location: "."

credentials:
  - name: cachix_token
    sources: [env:CACHIX_AUTH_TOKEN]
    scope: [ssh_keys]
`)
	if _, err := LoadConfiguration(initFile, types.Flags{}); err != nil {
		t.Fatalf("configuration loading resolved a scoped credential: %v", err)
	}
}
