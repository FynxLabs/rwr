package types

import "strings"

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

// showSecrets records whether the operator asked to see secret values, via
// --show-secrets. It is off unless explicitly enabled.
var showSecrets bool

// SetShowSecrets enables or disables printing secret values in logs.
func SetShowSecrets(enabled bool) {
	showSecrets = enabled
}

// ShowSecrets reports whether secret values may be printed.
func ShowSecrets() bool {
	return showSecrets
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
