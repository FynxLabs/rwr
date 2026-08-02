package processors

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessRepositories adds or removes package manager repositories from blueprint data,
// with support for profile filtering and import resolution.
func ProcessRepositories(blueprintData []byte, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var repositoriesBlueprint types.RepositoriesData
	var err error

	log.Debugf("Processing repositories from blueprint")

	// Unmarshal the blueprint data
	err = helpers.DecodeBlueprintInto(blueprintData, format, types.BlueprintTypeRepositories,
		helpers.TreeSchemaVersion(initConfig), &repositoriesBlueprint)
	if err != nil {
		return fmt.Errorf("error unmarshaling repository blueprint: %w", err)
	}

	// Process imports and merge imported repositories
	blueprintDir := initConfig.Init.Location
	allRepositories, err := processRepositoryImports(repositoriesBlueprint.Repositories, blueprintDir, format, helpers.TreeSchemaVersion(initConfig))
	if err != nil {
		return fmt.Errorf("error processing repository imports: %w", err)
	}
	repositoriesBlueprint.Repositories = allRepositories

	// Filter repositories based on active profiles
	filteredRepositories := helpers.FilterByProfiles(repositoriesBlueprint.Repositories, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering repositories: %d total, %d matching active profiles %v",
		len(repositoriesBlueprint.Repositories), len(filteredRepositories), initConfig.Variables.Flags.Profiles)

	// Process the filtered repositories
	err = processRepositories(filteredRepositories, osInfo, initConfig)
	if err != nil {
		return fmt.Errorf("error processing repositories: %w", err)
	}

	return nil
}

func processRepositories(repositories []types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers using the InitProviders function which handles embedded providers
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Process each repository
	for _, repo := range repositories {
		log.Infof("Processing repository %s", repo.Name)
		log.Debugf("Repository definition: %s", repo.LogString())

		// Get provider for this repository
		provider, exists := system.GetProvider(repo.PackageManager)
		if !exists {
			return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
		}

		// Get repository config
		repoConfig := provider.Repository

		// Execute repository action steps
		var steps []types.ActionStep
		switch repo.Action {
		case "add":
			steps = repoConfig.Add.Steps
		case "remove":
			steps = repoConfig.Remove.Steps
		default:
			return fmt.Errorf("unsupported repository action: %s", repo.Action)
		}

		// Execute each step
		data := repositoryStepData(repo, repoConfig.Paths, provider)
		for _, rawStep := range steps {
			// The condition is evaluated before the rest of the step is
			// rendered: a skipped step is allowed to reference data this
			// repository does not carry, which is the whole reason it is
			// conditional.
			run, err := stepCondition(rawStep.Condition, data)
			if err != nil {
				return fmt.Errorf("error evaluating condition for %s repository step of %s: %w", repo.Action, repo.Name, err)
			}
			if !run {
				log.Debugf("Skipping %s step: condition %q is not met", rawStep.Action, rawStep.Condition)
				continue
			}

			step, err := renderActionStep(rawStep, data)
			if err != nil {
				return fmt.Errorf("error rendering %s repository step for %s: %w", repo.Action, repo.Name, err)
			}

			var cmd types.Command

			switch step.Action {
			case "exec", "command": // Support both "exec" and "command" action types
				cmd = types.Command{
					Exec:        step.Exec,
					Args:        step.Args,
					Elevated:    provider.Elevated,
					Interactive: helpers.ResolveInteractive(repo.Interactive, initConfig.Variables.Flags.Interactive),
					// chocolatey's --password and cargo's login token are accepted
					// only as arguments, so they cannot move to stdin. They can at
					// least be kept out of the debug and dry-run log lines, which
					// print the full argv.
					Secrets: repo.SecretValues(),
				}
			case "download":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would download file: %s -> %s", step.Source, step.Dest)
					continue
				}
				if err := system.DownloadFile(step.Source, step.Dest, provider.Elevated); err != nil {
					return fmt.Errorf("error downloading file: %w", err)
				}
				continue
			case "write":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would write file: %s", step.Dest)
					continue
				}
				if err := system.WriteToFile(step.Dest, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error writing file: %w", err)
				}
				continue
			case "append":
				path, err := repositoryFilePath(step.Path, repoConfig.Paths)
				if err != nil {
					return fmt.Errorf("error appending to repository file for %s: %w", repo.Name, err)
				}
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would append to file: %s", path)
					continue
				}
				if err := system.AppendToFile(path, step.Content, provider.Elevated); err != nil {
					return fmt.Errorf("error appending to %s: %w", path, err)
				}
				continue
			case "remove_line":
				path, err := repositoryFilePath(step.Path, repoConfig.Paths)
				if err != nil {
					return fmt.Errorf("error removing line from repository file for %s: %w", repo.Name, err)
				}
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would remove lines naming %s from file: %s", step.Match, path)
					continue
				}
				if err := system.RemoveLineFromFile(path, step.Match, provider.Elevated); err != nil {
					return fmt.Errorf("error removing line from %s: %w", path, err)
				}
				continue
			case "remove_section":
				path, err := repositoryFilePath(step.Path, repoConfig.Paths)
				if err != nil {
					return fmt.Errorf("error removing section from repository file for %s: %w", repo.Name, err)
				}
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would remove section [%s] from file: %s", step.Section, path)
					continue
				}
				if err := system.RemoveSectionFromFile(path, step.Section, provider.Elevated); err != nil {
					return fmt.Errorf("error removing section from %s: %w", path, err)
				}
				continue
			case "remove":
				if err := removeRepositoryPath(step.Path, repoConfig.Paths, provider.Elevated, initConfig.Variables.Flags.Debug); err != nil {
					return fmt.Errorf("error removing repository path for %s: %w", repo.Name, err)
				}
				continue
			case "copy":
				if system.IsDryRun() {
					log.Infof("[DRY-RUN] Would copy file: %s -> %s", step.Source, step.Dest)
					continue
				}
				if err := system.CopyFile(step.Source, step.Dest, provider.Elevated, osInfo); err != nil {
					return fmt.Errorf("error copying file: %w", err)
				}
				continue
			default:
				return fmt.Errorf("unsupported repository action step: %s", step.Action)
			}

			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				return fmt.Errorf("error executing repository step: %w", err)
			}
		}
	}

	// Run updates for all available providers
	available := system.GetAvailableProviders()
	for name, provider := range available {
		if provider.Commands.Update == "" {
			continue
		}

		log.Infof("Processing %s Updates", name)
		updateCmd := types.Command{
			Exec:     provider.BinPath,
			Args:     strings.Fields(provider.Commands.Update),
			Elevated: provider.Elevated,
		}

		if err := system.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Warnf("Error updating %s package lists: %v", name, err)
			continue
		}
	}

	return nil
}

// repositoryStepData is the value a provider's repository action steps are
// rendered against. Provider definitions are Go templates — apt writes
// "deb [arch={{ .Arch }} signed-by={{ .KeyPath }}] {{ .URL }} ..." — so every
// placeholder they may use has to have a key here, and anything else is a
// blueprint or provider defect rather than a value to silently drop.
func repositoryStepData(repo types.Repository, paths types.RepositoryPaths, provider *types.Provider) map[string]any {
	// Providers reference a key file, but RepositoryPaths.Keys names the
	// directory a distribution keeps keyrings in, so the per-repository file
	// name is derived from the repository name rather than configured twice.
	keyPath := ""
	if paths.Keys != "" {
		keyPath = filepath.Join(paths.Keys, repo.Name+".gpg")
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
		"TempKeyPath":    filepath.Join(os.TempDir(), repo.Name+".gpg"),
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

	return data
}

// repositoryPredicates are the boolean-ish values provider steps gate
// themselves on with `condition = "{{ .HasKey }}"`.
//
// They are derived here rather than configured, because a repository blueprint
// says what a repository is and the provider says what to do about it. A
// predicate a provider names but that is absent from this map is an error at
// the point the step would have run: the conditions used to decode into
// nothing, so every step ran and flatpak added its remote both --user and
// --system — silently skipping an underivable predicate would reintroduce the
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

// isLocalPath reports whether a repository URL names a file on this machine
// rather than something to fetch over the network.
func isLocalPath(url string) bool {
	if url == "" {
		return false
	}
	if strings.HasPrefix(url, "file://") {
		return true
	}
	scheme, _, found := strings.Cut(url, "://")
	return !found || scheme == ""
}

// localPath is the filesystem path a local repository URL names.
func localPath(url string) string {
	if !isLocalPath(url) {
		return ""
	}
	return system.ExpandPath(strings.TrimPrefix(url, "file://"))
}

// stepCondition reports whether a step's condition holds.
//
// Only an explicit truthy rendering runs the step. A condition that renders to
// anything else — including the empty string a nil or missing value produces —
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

// repositoryFilePath resolves the file an in-place edit step names and refuses
// any file the provider does not declare in [provider.repository.paths].
//
// It is the containment removeRepositoryPath applies, widened by one case: a
// declared path may itself be the file being edited, because /etc/pacman.conf
// and macports' sources.conf are files rather than directories.
func repositoryFilePath(path string, paths types.RepositoryPaths) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("step has no path")
	}

	resolved := filepath.Clean(system.ExpandPath(path))
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("refusing to edit %q: path is not absolute", path)
	}

	boundaries := repositoryBoundaries(paths)
	if len(boundaries) == 0 {
		return "", fmt.Errorf("refusing to edit %q: provider declares no repository paths", resolved)
	}

	for _, boundary := range boundaries {
		if resolved == boundary {
			return resolved, nil
		}
	}
	if withinAny(resolved, boundaries) {
		return resolved, nil
	}

	return "", fmt.Errorf("refusing to edit %q: outside the provider's repository paths %v", resolved, boundaries)
}

// removeRepositoryPath deletes one path named by a provider's "remove" step.
//
// The path is a provider template rendered against blueprint values, and the
// deletion runs with whatever privilege the provider declares — root, for every
// system package manager. So it is only carried out inside the directories the
// provider itself declares in [provider.repository.paths]: neither a provider
// definition nor a repository name may redirect a root-owned delete at
// /etc/passwd. Symlinked directory components are resolved before the check, so a
// planted link cannot lead the delete out of those directories either.
func removeRepositoryPath(path string, paths types.RepositoryPaths, elevated, debug bool) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("remove step has no path")
	}

	resolved := filepath.Clean(system.ExpandPath(path))
	if !filepath.IsAbs(resolved) {
		return fmt.Errorf("refusing to remove %q: path is not absolute", path)
	}

	boundaries := repositoryBoundaries(paths)
	if len(boundaries) == 0 {
		return fmt.Errorf("refusing to remove %q: provider declares no repository paths", resolved)
	}

	if !withinAny(resolved, boundaries) {
		return fmt.Errorf("refusing to remove %q: outside the provider's repository paths %v", resolved, boundaries)
	}

	realPath, err := realRemovalPath(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			// `rwr all` is expected to be run repeatedly, and a removal that has
			// already happened is the state the blueprint asked for.
			log.Debugf("Repository path already absent: %s", resolved)
			return nil
		}
		return fmt.Errorf("error resolving %q: %w", resolved, err)
	}

	if !withinAny(realPath, boundaries) {
		return fmt.Errorf("refusing to remove %q: %q is outside the provider's repository paths %v", resolved, realPath, boundaries)
	}

	if system.IsDryRun() {
		log.Infof("[DRY-RUN] Would remove file: %s", resolved)
		return nil
	}

	if elevated {
		// Sources lists and keyrings are root-owned, so the delete needs the same
		// elevation the writes that created them used. "-f" keeps it idempotent.
		return system.RunCommand(types.Command{
			Exec:     "rm",
			Args:     []string{"-f", "--", resolved},
			Elevated: true,
		}, debug)
	}

	if err := os.Remove(resolved); err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Repository path already absent: %s", resolved)
			return nil
		}
		return err
	}

	log.Infof("Removed repository file: %s", resolved)
	return nil
}

// repositoryBoundaries returns the directories a remove step may act inside.
func repositoryBoundaries(paths types.RepositoryPaths) []string {
	var boundaries []string
	for _, declared := range []string{paths.Sources, paths.Keys, paths.Config} {
		if declared == "" {
			continue
		}
		expanded := filepath.Clean(system.ExpandPath(declared))
		if !filepath.IsAbs(expanded) {
			continue
		}
		if evaluated, err := filepath.EvalSymlinks(expanded); err == nil {
			boundaries = append(boundaries, evaluated)
		}
		boundaries = append(boundaries, expanded)
	}
	return boundaries
}

// realRemovalPath resolves symlinked directory components of path, leaving the
// final element alone: removing a symlink is meant to remove the link itself.
func realRemovalPath(path string) (string, error) {
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

// withinAny reports whether path sits strictly under one of the boundaries. The
// boundary directory itself is not removable: no provider asks for it, and
// deleting /etc/apt/sources.list.d wholesale is never what a blueprint meant.
func withinAny(path string, boundaries []string) bool {
	for _, boundary := range boundaries {
		if strings.HasPrefix(path, boundary+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// renderActionStep renders every templated field of a step. Only a literal
// "{{ .URL }}" argument was ever substituted before, so an apt repository add
// wrote a file literally named "{{ .SourcesPath }}/{{ .Name }}.list" holding
// "deb [arch={{ .Arch }} ...]".
func renderActionStep(step types.ActionStep, data map[string]any) (types.ActionStep, error) {
	rendered := step

	for _, field := range []*string{&rendered.Source, &rendered.Dest, &rendered.Exec, &rendered.Content, &rendered.Path, &rendered.Match, &rendered.Section} {
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

func processRepositoryImports(items []types.Repository, blueprintDir string, format string, treeVersion int) ([]types.Repository, error) {
	return helpers.ResolveImports(items, blueprintDir,
		func(item types.Repository) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Repository, error) {
			var d types.RepositoriesData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypeRepositories, treeVersion, &d); err != nil {
				return nil, err
			}
			return d.Repositories, nil
		}, format)
}
