package types

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsSecretConfigKey(t *testing.T) {
	secret := []string{
		"repository.gh_api_token",
		"repository.ssh_private_key",
		"REPOSITORY.GH_API_TOKEN",
	}
	notSecret := []string{
		"repository.init-file",
		"repository.url",
		"rwr.profiles",
		"log.level",
	}

	for _, key := range secret {
		if !IsSecretConfigKey(key) {
			t.Errorf("IsSecretConfigKey(%q) = false, want true", key)
		}
	}
	for _, key := range notSecret {
		if IsSecretConfigKey(key) {
			t.Errorf("IsSecretConfigKey(%q) = true, want false", key)
		}
	}
}

func TestRedact(t *testing.T) {
	SetShowSecrets(false)
	defer SetShowSecrets(false)

	if got := Redact("ghp_supersecret"); got != RedactedPlaceholder {
		t.Errorf("Redact = %q, want %q", got, RedactedPlaceholder)
	}
	if got := Redact(""); got != "" {
		t.Errorf("Redact(\"\") = %q, want empty (nothing to hide)", got)
	}

	SetShowSecrets(true)
	if got := Redact("ghp_supersecret"); got != "ghp_supersecret" {
		t.Errorf("with --show-secrets, Redact = %q, want the value", got)
	}
}

// Flags reached debug logs via %+v, which printed the token and the key. The
// String method is what stops that, so it has to actually hide them.
func TestFlagsString_RedactsCredentials(t *testing.T) {
	SetShowSecrets(false)
	defer SetShowSecrets(false)

	flags := Flags{
		GHAPIToken: "ghp_thisisatoken",
		SSHKey:     "LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0=",
		LogLevel:   "debug",
	}

	rendered := flags.String()

	if strings.Contains(rendered, "ghp_thisisatoken") {
		t.Errorf("Flags.String leaked the GitHub token: %s", rendered)
	}
	if strings.Contains(rendered, "LS0tLS1CRUdJTiBPUEVOU1NI") {
		t.Errorf("Flags.String leaked the SSH key: %s", rendered)
	}
	if !strings.Contains(rendered, RedactedPlaceholder) {
		t.Errorf("Flags.String should mark redacted fields: %s", rendered)
	}
	// Non-secret fields still need to be useful for debugging.
	if !strings.Contains(rendered, "debug") {
		t.Errorf("Flags.String dropped non-secret fields: %s", rendered)
	}
}

func TestFlagsString_ShowsCredentialsWhenAsked(t *testing.T) {
	SetShowSecrets(true)
	defer SetShowSecrets(false)

	flags := Flags{GHAPIToken: "ghp_thisisatoken"}

	if !strings.Contains(flags.String(), "ghp_thisisatoken") {
		t.Error("with --show-secrets, Flags.String should show the token")
	}
}

// Templates are rendered from blueprint files cloned out of git repositories, and
// the result is written wherever that blueprint says. Credentials must not be
// reachable from template scope at all.
func TestFlagsToMap_ExcludesCredentials(t *testing.T) {
	flags := Flags{
		GHAPIToken: "ghp_thisisatoken",
		SSHKey:     "a-private-key",
		LogLevel:   "debug",
	}

	m := flags.ToMap()

	for _, key := range []string{"ghAPIToken", "sshKey"} {
		if _, present := m[key]; present {
			t.Errorf("Flags.ToMap exposes %q to blueprint templates", key)
		}
	}

	for _, value := range m {
		if s, ok := value.(string); ok {
			if s == "ghp_thisisatoken" || s == "a-private-key" {
				t.Errorf("Flags.ToMap exposes a credential value: %q", s)
			}
		}
	}

	if m["logLevel"] != "debug" {
		t.Errorf("Flags.ToMap dropped a non-secret field: %v", m["logLevel"])
	}
}

// The real log lines print the whole InitConfig or Variables, not Flags directly
// (initialize.go and root.go both do), so redaction has to survive nesting.
func TestNestedStructsDoNotLeakCredentials(t *testing.T) {
	SetShowSecrets(false)
	defer SetShowSecrets(false)

	initConfig := InitConfig{}
	initConfig.Variables.Flags = Flags{
		GHAPIToken: "ghp_thisisatoken",
		SSHKey:     "a-private-key",
	}

	rendered := []string{
		fmt.Sprintf("%v", initConfig),
		fmt.Sprintf("%+v", initConfig),
		fmt.Sprintf("%v", initConfig.Variables),
		fmt.Sprintf("%+v", initConfig.Variables),
	}

	for _, s := range rendered {
		if strings.Contains(s, "ghp_thisisatoken") {
			t.Errorf("nested formatting leaked the GitHub token: %s", s)
		}
		if strings.Contains(s, "a-private-key") {
			t.Errorf("nested formatting leaked the SSH key: %s", s)
		}
	}
}

// Credentials are withheld from templates by default, but a blueprint that
// genuinely needs one — writing a .netrc, configuring gh — can opt in by name.
func TestFlagsToMap_ExposesOnlyOptedInCredentials(t *testing.T) {
	defer SetExposedCredentials(nil)

	flags := Flags{GHAPIToken: "ghp_thisisatoken", SSHKey: "a-private-key"}

	SetExposedCredentials(nil)
	if _, present := flags.ToMap()["ghAPIToken"]; present {
		t.Error("no opt-in, yet the token is in template scope")
	}

	// Opting into one credential must not expose the other.
	SetExposedCredentials([]string{"gh_api_token"})
	m := flags.ToMap()
	if m["ghAPIToken"] != "ghp_thisisatoken" {
		t.Errorf("opted-in token missing from template scope: %v", m["ghAPIToken"])
	}
	if _, present := m["sshKey"]; present {
		t.Error("opting into the token also exposed the ssh key")
	}

	SetExposedCredentials([]string{"ssh_private_key"})
	m = flags.ToMap()
	if m["sshKey"] != "a-private-key" {
		t.Errorf("opted-in ssh key missing from template scope: %v", m["sshKey"])
	}
	if _, present := m["ghAPIToken"]; present {
		t.Error("opting into the ssh key also exposed the token")
	}
}

// The opt-in accepts either the full viper key or the bare name, since the init
// file and the config key spell it differently.
func TestIsCredentialExposed_AcceptsEitherSpelling(t *testing.T) {
	defer SetExposedCredentials(nil)

	SetExposedCredentials([]string{"repository.gh_api_token"})
	if !IsCredentialExposed("gh_api_token") {
		t.Error("a full viper key should match the bare name")
	}
	if !IsCredentialExposed("repository.gh_api_token") {
		t.Error("a full viper key should match itself")
	}
	if IsCredentialExposed("ssh_private_key") {
		t.Error("an unrelated credential should stay withheld")
	}
}
