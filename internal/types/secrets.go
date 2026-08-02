package types

import (
	"sort"
	"strings"
)

// SecretConfigKeys are the viper config keys holding credentials.
//
// These are the values that must not leave the rwr process: not into the
// environment of the commands it spawns, not into blueprint template scope, and
// not into logs. Blueprints are cloned from git repositories and scripts run with
// the environment rwr hands them, so anything reachable from either is readable
// by whoever wrote the blueprint.
var SecretConfigKeys = []string{
	"repository.gh_api_token",
	"repository.ssh_private_key",
}

// RedactedPlaceholder is substituted wherever a secret would otherwise be shown.
const RedactedPlaceholder = "[redacted]"

// IsSecretConfigKey reports whether a viper config key holds a credential.
func IsSecretConfigKey(key string) bool {
	for _, secret := range SecretConfigKeys {
		if strings.EqualFold(key, secret) {
			return true
		}
	}
	return false
}

// exposedCredentials names the credentials the init file has opted into sharing
// with blueprint templates and spawned commands.
//
// Credentials are withheld by default because blueprints are cloned from git
// repositories and scripts inherit rwr's environment, so anything reachable from
// either is readable by whoever wrote the blueprint. But some blueprints
// legitimately need a token — writing a .netrc, configuring gh, calling the
// GitHub API from a script — so this is opt-in rather than unavailable.
var exposedCredentials = map[string]bool{}

// SetExposedCredentials records which credential keys the operator opted into.
// Names may be given as viper keys ("repository.gh_api_token") or bare
// ("gh_api_token").
func SetExposedCredentials(keys []string) {
	exposedCredentials = make(map[string]bool, len(keys))
	for _, key := range keys {
		exposedCredentials[normaliseCredentialKey(key)] = true
	}
}

// IsCredentialExposed reports whether a credential key was opted into.
func IsCredentialExposed(key string) bool {
	return exposedCredentials[normaliseCredentialKey(key)]
}

// ExposedCredentials returns the opted-in keys, for logging.
func ExposedCredentials() []string {
	out := make([]string, 0, len(exposedCredentials))
	for key := range exposedCredentials {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// normaliseCredentialKey reduces a viper key or bare name to a common form, so
// "repository.gh_api_token" and "gh_api_token" mean the same thing.
func normaliseCredentialKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		key = key[idx+1:]
	}
	return key
}

// showSecrets records whether the operator asked to see secret values, via
// --show-secrets. It is off unless explicitly enabled.
var showSecrets bool

// SetShowSecrets enables or disables printing secret values in logs.
func SetShowSecrets(enabled bool) {
	showSecrets = enabled
}

// Redact returns value unless the operator asked to see secrets.
//
// The escape hatch exists because "is rwr even reading my token?" is a real
// question with no other answer — but it has to be asked for deliberately rather
// than being the default any time debug logging is on.
func Redact(value string) string {
	if value == "" {
		return ""
	}
	if showSecrets {
		return value
	}
	return RedactedPlaceholder
}
