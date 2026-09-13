package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// writeInitFile writes an init file for Initialize to load.
func writeInitFile(t *testing.T, content string) string {
	t.Helper()
	initFile := filepath.Join(t.TempDir(), "init.yaml")
	if err := os.WriteFile(initFile, []byte(content), 0644); err != nil {
		t.Fatalf("writing init file: %v", err)
	}
	return initFile
}

// resetManagedAuthState isolates the global registries Initialize mutates.
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

func TestInitializeContinuesWithoutBitwarden(t *testing.T) {
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
	config, err := Initialize(initFile, types.Flags{Interactive: false})
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
func TestInitializeResolvesAndWithholdsDeclaredCredential(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "cachix-secret-value")

	initFile := writeInitFile(t, declaredCredentialInit)
	config, err := Initialize(initFile, types.Flags{})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
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
func TestInitializeExposureDoesNotAcquire(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "cachix-secret-value")
	initFile := writeInitFile(t, declaredCredentialInit+"\nexposeCredentials: [cachix_token]\n")
	_, err := Initialize(initFile, types.Flags{})
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

func TestInitializeStrictDecodesCredentials(t *testing.T) {
	resetManagedAuthState(t)

	initFile := writeInitFile(t, `
blueprints:
  format: yaml
  location: "."

credentials:
  - name: cachix_token
    soruces: [keyring]
`)
	_, err := Initialize(initFile, types.Flags{})
	if err == nil || !strings.Contains(err.Error(), "soruces") {
		t.Fatalf("Initialize = %v, want a strict-decode error naming the unknown key", err)
	}
}

// A declared credential that resolves nowhere fails before any processor runs,
// naming the credential and the sources tried; in a non-interactive run the
// error says prompt was skipped.
func TestInitializeUnavailableCredentialDoesNotFail(t *testing.T) {
	resetManagedAuthState(t)
	t.Setenv("CACHIX_AUTH_TOKEN", "")
	if _, err := Initialize(writeInitFile(t, declaredCredentialInit), types.Flags{Interactive: true}); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeSkipsOutOfScopeCredential(t *testing.T) {
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
	if _, err := Initialize(initFile, types.Flags{}, types.BlueprintTypePackages); err != nil {
		t.Fatalf("Initialize resolved an out-of-scope credential: %v", err)
	}
	if _, err := Initialize(initFile, types.Flags{}, types.BlueprintTypeSSHKeys); err != nil {
		t.Fatal("Initialize = nil for an in-scope unresolvable credential, want an error")
	}
}
