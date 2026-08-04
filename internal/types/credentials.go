package types

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

// CredentialSpec is one entry of the init file's `credentials:` section: a named
// credential the tree's blueprints need, and the ordered places to look for it.
//
// Declarations live only in the init file the operator points rwr at - a
// blueprint can reference a declared credential but never declare one - and
// exposure to blueprints still requires the operator's exposeCredentials opt-in.
type CredentialSpec struct {
	// Name is the credential's identity everywhere: in exposeCredentials, in
	// template scope ({{ .Credentials.<name> }}), in the exported env name
	// (RWR_CRED_<NAME>), and in the keyring entry.
	Name string `mapstructure:"name" yaml:"name" json:"name" toml:"name"`
	// Description is shown when prompting for the value.
	Description string `mapstructure:"description,omitempty" yaml:"description,omitempty" json:"description,omitempty" toml:"description,omitempty"`
	// Sources is the ordered list of places to look: `env:<VAR>`, `keyring`,
	// `prompt`. First source that yields a value wins. Empty means the default
	// order [env:RWR_CRED_<NAME>, keyring, prompt].
	Sources []string `mapstructure:"sources,omitempty" yaml:"sources,omitempty" json:"sources,omitempty" toml:"sources,omitempty"`
	// Scope limits which surfaces see the credential even after exposure:
	// "scripts" (spawned-command env), "templates" (template rendering), or a
	// processor name (the credential is only resolved when that processor is in
	// the run). Empty reproduces exposeCredentials' full reach.
	Scope []string `mapstructure:"scope,omitempty" yaml:"scope,omitempty" json:"scope,omitempty" toml:"scope,omitempty"`
}

// Surfaces a credential's scope can name, beyond processor names.
const (
	CredentialSurfaceScripts   = "scripts"
	CredentialSurfaceTemplates = "templates"
)

// credentialNamePattern is deliberately narrow: the name becomes an env var
// segment and a template key, so it has to be a lowercase identifier.
var credentialNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// DecodeCredentialSpecs strictly decodes the raw `credentials:` section as read
// by viper. Viper's own Unmarshal ignores unknown keys, so a typo like
// `soruces:` would silently become the default source order - an error naming
// the key is the only honest answer.
func DecodeCredentialSpecs(raw interface{}) ([]CredentialSpec, error) {
	if raw == nil {
		return nil, nil
	}
	var specs []CredentialSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &specs,
		ErrorUnused: true,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(raw); err != nil {
		return nil, fmt.Errorf("credentials section: %w", err)
	}
	return specs, ValidateCredentialSpecs(specs)
}

// ValidateCredentialSpecs rejects a declaration rwr cannot honor: a name that
// cannot become an env var or template key, a source outside the vocabulary, a
// scope naming no known surface or processor, or a collision with a built-in.
func ValidateCredentialSpecs(specs []CredentialSpec) error {
	seen := map[string]bool{}
	for _, spec := range specs {
		if !credentialNamePattern.MatchString(spec.Name) {
			return fmt.Errorf("credential name %q is invalid: names are lowercase identifiers ([a-z][a-z0-9_]*)", spec.Name)
		}
		if seen[spec.Name] {
			return fmt.Errorf("credential %q is declared twice", spec.Name)
		}
		seen[spec.Name] = true
		if isBuiltinCredential(spec.Name) {
			return fmt.Errorf("credential %q is built in and cannot be redeclared", spec.Name)
		}
		for _, source := range spec.Sources {
			if err := validateCredentialSource(source); err != nil {
				return fmt.Errorf("credential %q: %w", spec.Name, err)
			}
		}
		for _, scope := range spec.Scope {
			if err := validateCredentialScope(scope); err != nil {
				return fmt.Errorf("credential %q: %w", spec.Name, err)
			}
		}
	}
	return nil
}

func validateCredentialSource(source string) error {
	switch {
	case source == "keyring" || source == "prompt":
		return nil
	case strings.HasPrefix(source, "env:"):
		if strings.TrimPrefix(source, "env:") == "" {
			return fmt.Errorf("source %q names no environment variable", source)
		}
		return nil
	default:
		return fmt.Errorf("unknown source %q (valid: env:<VAR>, keyring, prompt)", source)
	}
}

func validateCredentialScope(scope string) error {
	switch scope {
	case CredentialSurfaceScripts, CredentialSurfaceTemplates,
		BlueprintTypePackages, BlueprintTypeRepositories, BlueprintTypeFiles,
		BlueprintTypeGit, BlueprintTypeSSHKeys, BlueprintTypeFonts,
		BlueprintTypeUsers, BlueprintTypeConfiguration, BlueprintTypeServices,
		BlueprintTypeBootstrap:
		return nil
	default:
		return fmt.Errorf("unknown scope %q (valid: scripts, templates, or a processor name)", scope)
	}
}

// builtinCredentialNames are the two credentials rwr always manages, implicit
// declarations so existing behavior is a special case rather than a parallel
// system. SecretConfigKeys stays as the bridge from their viper keys.
var builtinCredentialNames = []string{"gh_api_token", "ssh_private_key"}

func isBuiltinCredential(name string) bool {
	for _, builtin := range builtinCredentialNames {
		if name == builtin {
			return true
		}
	}
	return false
}

// The credential registry: the declared set (scopes) and the values resolved
// for this run. Values live only here - never in viper, never in a struct a
// debug dump can reach - and leave only through the exposure-gated accessors
// below.
var (
	credentialScopes = map[string][]string{}
	credentialValues = map[string]string{}
)

// RegisterCredentials records the declared set for this run, replacing any
// previous registration (and dropping its values, so tests are isolated).
func RegisterCredentials(specs []CredentialSpec) {
	credentialScopes = make(map[string][]string, len(specs))
	credentialValues = map[string]string{}
	for _, spec := range specs {
		credentialScopes[spec.Name] = spec.Scope
	}
}

// IsManagedCredential reports whether name is a built-in or declared credential.
func IsManagedCredential(name string) bool {
	name = normaliseCredentialKey(name)
	if isBuiltinCredential(name) {
		return true
	}
	_, ok := credentialScopes[name]
	return ok
}

// SetCredentialValue records a resolved value. Only resolution may call this;
// everything downstream reads through the gated accessors.
func SetCredentialValue(name, value string) {
	credentialValues[normaliseCredentialKey(name)] = value
}

// CredentialValue returns a resolved value ungated, for rwr's own use (rwr
// doing the credentialed work itself is always allowed; handing the value to a
// blueprint is what the gates below are for).
func CredentialValue(name string) (string, bool) {
	value, ok := credentialValues[normaliseCredentialKey(name)]
	return value, ok
}

// CredentialScope returns the declared scope for a credential (empty for
// built-ins and unscoped declarations, meaning everything exposure reaches).
func CredentialScope(name string) []string {
	return credentialScopes[normaliseCredentialKey(name)]
}

// credentialInSurface reports whether the credential's scope admits a surface.
// An empty scope admits every surface; a scope listing only processor names
// restricts when the credential is resolved, not where it may appear.
func credentialInSurface(name, surface string) bool {
	scope := credentialScopes[normaliseCredentialKey(name)]
	if len(scope) == 0 {
		return true
	}
	restricted := false
	for _, entry := range scope {
		if entry == surface {
			return true
		}
		if entry == CredentialSurfaceScripts || entry == CredentialSurfaceTemplates {
			restricted = true
		}
	}
	return !restricted
}

// TemplateCredentials returns the values blueprint templates may render as
// {{ .Credentials.<name> }}: resolved, opted into via exposeCredentials, and
// in a scope that admits templates.
func TemplateCredentials() map[string]interface{} {
	out := map[string]interface{}{}
	for name, value := range credentialValues {
		if IsCredentialExposed(name) && credentialInSurface(name, CredentialSurfaceTemplates) {
			out[name] = value
		}
	}
	return out
}

// ExportedCredentialEnv returns the RWR_CRED_<NAME> environment entries for
// spawned commands: resolved, opted into via exposeCredentials, and in a scope
// that admits scripts. Env is the only export surface - a value must never be
// placed in argv, where ps can read it.
func ExportedCredentialEnv() map[string]string {
	out := map[string]string{}
	for name, value := range credentialValues {
		if IsCredentialExposed(name) && credentialInSurface(name, CredentialSurfaceScripts) {
			out["RWR_CRED_"+strings.ToUpper(name)] = value
		}
	}
	return out
}

// TemplateCredentialNames returns the exposed template-scope names, for the
// debug dump of template variables: the names are loggable, the values are not.
func TemplateCredentialNames() []string {
	names := make([]string, 0, len(credentialValues))
	for name := range TemplateCredentials() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
