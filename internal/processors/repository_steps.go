// The step machinery: rendering a provider's declared repository steps
// into executable commands - the template field namespace, the condition
// predicates, and the per-field rendering.

package processors

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fynxlabs/rwr/internal/types"
)

// repositoryStepData is the value a provider's repository action steps are
// rendered against. Provider definitions are Go templates - apt writes
// "deb [arch={{ .Arch }} signed-by={{ .KeyPath }}] {{ .URL }} ..." - so every
// placeholder they may use has to have a key here, and anything else is a
// blueprint or provider defect rather than a value to silently drop.
func repositoryStepData(repo types.Repository, paths types.RepositoryPaths, provider *types.Provider) (map[string]any, error) {
	// Providers reference a key file, but RepositoryPaths.Keys names the
	// directory a distribution keeps keyrings in, so the per-repository file
	// name is derived from the repository name rather than configured twice.
	keyPath := ""
	if paths.Keys != "" {
		keyPath = filepath.Join(paths.Keys, repo.Name+".gpg")
	}

	tempDir, err := repositoryTempDir()
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"Name":           repo.Name,
		"URL":            repo.URL,
		"KeyURL":         repo.KeyURL,
		"Channel":        repo.Channel,
		"Component":      repo.Component,
		"Arch":           repo.Arch,
		"Repository":     repo.Repository,
		"PackageManager": repo.PackageManager,
		"Action":         repo.Action,
		"SourcesPath":    paths.Sources,
		"KeysPath":       paths.Keys,
		"ConfigPath":     paths.Config,
		"KeyPath":        keyPath,
		"KeySha256":      repo.KeySha256,
		"TempKeyPath":    filepath.Join(tempDir, repo.Name+".gpg"),
		"KeyID":          repo.KeyID,
		// The local file a snap or a GNOME extension is installed from. Only
		// the steps gated on IsLocalFile/IsLocalSnap use it, and those are the
		// steps whose URL names a file on this machine.
		"Path":        localPath(repo.URL),
		"Description": repo.Description,
		"OverlayPath": repo.OverlayPath,
		"SyncType":    repo.SyncType,
		"SHA256":      repo.SHA256,
		"ProxyURL":    repo.ProxyURL,
		"UUID":        repo.UUID,
		"ExtensionID": repo.ExtensionID,
		"Interface":   repo.Interface,
		"Slot":        repo.Slot,

		// Not a derived predicate: whether removing a GNOME extension also
		// resets its settings is operator intent, straight from the blueprint.
		"ResetSettings": repo.ResetSettings,

		// Credentials reach the provider's argv because that is the only way
		// chocolatey and cargo can be told about a private source. They are kept out
		// of everything rwr prints itself; see the note on types.Repository.
		"Username": repo.Username,
		"Password": repo.Password,
		"Token":    repo.Token,
	}

	for name, value := range repositoryPredicates(repo, provider) {
		data[name] = value
	}

	return data, nil
}

// repositoryPredicates are the boolean-ish values provider steps gate
// themselves on with `condition = "{{ .HasKey }}"`.
//
// They are derived here rather than configured, because a repository blueprint
// says what a repository is and the provider says what to do about it. A
// predicate a provider names but that is absent from this map is an error at
// the point the step would have run: the conditions used to decode into
// nothing, so every step ran and flatpak added its remote both --user and
// --system - silently skipping an underivable predicate would reintroduce the
// same class of bug in the other direction.
func repositoryPredicates(repo types.Repository, provider *types.Provider) map[string]any {
	localURL := isLocalPath(repo.URL)

	return map[string]any{
		"HasKey":            repo.KeyURL != "" || repo.KeyID != "",
		"HasInterfaces":     repo.Interface != "",
		"HasSlot":           repo.Slot != "",
		"HasProxy":          repo.ProxyURL != "",
		"HasToken":          repo.Token != "",
		"HasAuthentication": repo.Username != "" || repo.Password != "",
		"RequiresAuth":      repo.Username != "" || repo.Password != "" || repo.Token != "",

		// A cargo registry other than crates.io is the one that has to be
		// written into config.toml with an index URL.
		"IsCustomRegistry": repo.URL != "",

		// An overlay is a nix/emerge repository grafted onto the distribution's
		// own tree, which is what an overlay path or a pinned tarball digest
		// identifies.
		"IsOverlay": repo.OverlayPath != "" || repo.SHA256 != "",

		// Portage's own repository, the only one `emerge --sync` refreshes.
		"IsMainRepo": repo.Name == "gentoo",

		// A source rwr already has on disk rather than one to fetch.
		"IsLocalFile": localURL,
		"IsLocalSnap": localURL,
		"IsSnapStore": !localURL,

		// A provider that does not elevate manages the invoking user's own
		// installation; the elevated one manages the system's.
		"UserMode": provider == nil || !provider.Elevated,
	}
}

// stepCondition reports whether a step's condition holds.
//
// Only an explicit truthy rendering runs the step. A condition that renders to
// anything else - including the empty string a nil or missing value produces -
// means the provider did not ask for this step on this repository.
func stepCondition(condition string, data map[string]any) (bool, error) {
	if strings.TrimSpace(condition) == "" {
		return true, nil
	}

	rendered, err := renderStepField(condition, data)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(rendered)) {
	case "true", "1", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// renderActionStep renders every templated field of a step. Only a literal
// "{{ .URL }}" argument was ever substituted before, so an apt repository add
// wrote a file literally named "{{ .SourcesPath }}/{{ .Name }}.list" holding
// "deb [arch={{ .Arch }} ...]".
func renderActionStep(step types.ActionStep, data map[string]any) (types.ActionStep, error) {
	rendered := step

	for _, field := range []*string{&rendered.Source, &rendered.Dest, &rendered.Exec, &rendered.Content, &rendered.Path, &rendered.Match, &rendered.Section, &rendered.Sha256} {
		value, err := renderStepField(*field, data)
		if err != nil {
			return types.ActionStep{}, err
		}
		*field = value
	}

	if step.Args != nil {
		rendered.Args = make([]string, len(step.Args))
		for i, arg := range step.Args {
			value, err := renderStepField(arg, data)
			if err != nil {
				return types.ActionStep{}, err
			}
			rendered.Args[i] = value
		}
	}

	return rendered, nil
}

// renderStepField renders one field with missingkey=error, matching
// helpers.ResolveTemplate: a placeholder rwr cannot fill is reported rather than
// rendered as "<no value>" and then used as a path, a URL or a package name.
func renderStepField(field string, data map[string]any) (string, error) {
	if !strings.Contains(field, "{{") {
		return field, nil
	}

	tmpl, err := template.New("step").Option("missingkey=error").Parse(field)
	if err != nil {
		return "", fmt.Errorf("error parsing template %q: %w", field, err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("error rendering template %q: %w", field, err)
	}

	return out.String(), nil
}
