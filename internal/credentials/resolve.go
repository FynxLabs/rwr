package credentials

import (
	"fmt"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// Options carries the run context resolution needs.
type Options struct {
	// Interactive mirrors --interactive; prompting also requires a real TTY.
	Interactive bool
	// Selected names the blueprint types this run executes; empty means all. A
	// credential scoped to processors none of which are selected is not
	// resolved - no point prompting for a value nothing will read.
	Selected []string
}

// Resolve resolves every declared credential up front, first source wins, and
// records the values in the types registry. All of them resolve before any
// processor runs: failing at minute 20 inside a script is worse than failing
// at second 1. A credential that resolves from no source fails the run with an
// error naming it and the sources tried.
func Resolve(specs []types.CredentialSpec, opts Options) error {
	for _, spec := range specs {
		if !scopeSelected(spec.Scope, opts.Selected) {
			log.Debugf("Not resolving credential %q: its scope %v matches no selected processor", spec.Name, spec.Scope)
			continue
		}
		value, err := resolveOne(spec, opts)
		if err != nil {
			return err
		}
		types.SetCredentialValue(spec.Name, value)
	}
	return nil
}

// defaultSources is the order used when a declaration omits `sources`: env
// first so CI can inject without prompting, keyring before prompt so an
// interactive answer given once (and saved) is not re-asked.
func defaultSources(name string) []string {
	return []string{"env:RWR_CRED_" + strings.ToUpper(name), "keyring", "prompt"}
}

func resolveOne(spec types.CredentialSpec, opts Options) (string, error) {
	sources := spec.Sources
	if len(sources) == 0 {
		sources = defaultSources(spec.Name)
	}

	var tried, skipped []string
	for _, source := range sources {
		switch {
		case strings.HasPrefix(source, "env:"):
			envVar := strings.TrimPrefix(source, "env:")
			if value := os.Getenv(envVar); value != "" {
				log.Debugf("Credential %q resolved from %s: %s", spec.Name, source, types.Redact(value))
				return value, nil
			}
			tried = append(tried, source)

		case source == "keyring":
			if value, ok := FromKeyring(spec.Name); ok {
				log.Debugf("Credential %q resolved from the keyring: %s", spec.Name, types.Redact(value))
				return value, nil
			}
			tried = append(tried, source)

		case source == "prompt":
			if !opts.Interactive || !stdinIsTerminal() {
				skipped = append(skipped, "prompt")
				continue
			}
			value, err := promptForCredential(spec.Name, spec.Description)
			if err != nil {
				return "", err
			}
			offerKeyringSave(spec.Name, value)
			return value, nil

		default:
			// Validation rejected unknown sources at decode time; reaching here
			// means a caller bypassed it.
			return "", fmt.Errorf("credential %q: unknown source %q", spec.Name, source)
		}
	}

	var parts []string
	if len(tried) > 0 {
		parts = append(parts, "tried: "+strings.Join(tried, ", "))
	}
	if len(skipped) > 0 {
		parts = append(parts, "prompt skipped: non-interactive run")
	}
	return "", fmt.Errorf("credential %q resolved from no source (%s)", spec.Name, strings.Join(parts, "; "))
}

// scopeSelected reports whether a credential's scope intersects the selected
// processors. Surface entries (scripts, templates) map to the processors that
// realize them; an empty scope or an empty selection always matches.
func scopeSelected(scope, selected []string) bool {
	if len(scope) == 0 || len(selected) == 0 {
		return true
	}
	for _, entry := range scope {
		processor := entry
		// Template rendering is realized by the files processor.
		if entry == types.CredentialSurfaceTemplates {
			processor = types.BlueprintTypeFiles
		}
		for _, name := range selected {
			if name == processor {
				return true
			}
		}
	}
	return false
}

// ResolveBuiltins records the two built-in credentials with the registry so
// they get the same managed treatment as declared ones. Their sources are the
// pre-existing ones - flag/config (already merged into Flags by cobra/viper
// binding), GITHUB_TOKEN, and now the keyring - and unlike declared
// credentials they are optional: a run that never needs a GitHub token must
// not fail, or prompt, because none is configured.
func ResolveBuiltins(flags *types.Flags) {
	switch {
	case flags.GHAPIToken != "":
		types.SetCredentialValue("gh_api_token", flags.GHAPIToken)
	case os.Getenv("GITHUB_TOKEN") != "":
		types.SetCredentialValue("gh_api_token", os.Getenv("GITHUB_TOKEN"))
	default:
		if value, ok := FromKeyring("gh_api_token"); ok {
			log.Debugf("GitHub token resolved from the OS keyring")
			types.SetCredentialValue("gh_api_token", value)
			// Downstream consumers (git auth, ssh-key upload) read the flag
			// field, so a keyring token flows through the same path.
			flags.GHAPIToken = value
		}
	}
	if flags.SSHKey != "" {
		types.SetCredentialValue("ssh_private_key", flags.SSHKey)
	} else if value, ok := FromKeyring("ssh_private_key"); ok {
		log.Debugf("SSH private key resolved from the OS keyring")
		types.SetCredentialValue("ssh_private_key", value)
		// Git authentication reads the flag field, so keyring material flows
		// through the same path as --ssh-key and the config fallback.
		flags.SSHKey = value
	}
}
