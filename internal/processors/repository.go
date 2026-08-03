package processors

import (
	"fmt"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessRepositories adds or removes package manager repositories from blueprint data,
// with support for profile filtering and import resolution.
func ProcessRepositories(blueprintData []byte, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
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

	// One repository failing does not stop the rest: the failure goes to the
	// ledger, which puts it in the run's exit code, and processing continues.
	track := newProgress(types.BlueprintTypeRepositories)
	for _, repo := range repositories {
		track.expect(repo.PackageManager, 1)
	}
	declared := make(map[string]bool, len(repositories))
	for _, repo := range repositories {
		log.Infof("Processing repository %s", repo.Name)
		log.Debugf("Repository definition: %s", repo.LogString())
		declared[repo.PackageManager] = true

		started := time.Now()
		switch err := processRepository(repo, osInfo, initConfig); {
		case err != nil:
			recordFailure("repositories", repo.Name, err)
			track.item(repo.PackageManager, repo.Name, repo.Action, types.StatusFailed, err.Error(), time.Since(started))
		case system.IsDryRun():
			track.item(repo.PackageManager, repo.Name, repo.Action, types.StatusPlanned, "dry-run", 0)
		default:
			track.item(repo.PackageManager, repo.Name, repo.Action, types.StatusOK, "", time.Since(started))
		}
	}

	return runRepositoryUpdates(initConfig, declared)
}

// validateRepositoryName refuses names that would escape the paths derived
// from them. repo.Name is blueprint-supplied and is joined into KeyPath,
// TempKeyPath and provider-templated file names ("{{ .SourcesPath }}/
// {{ .Name }}.list"), all written with the provider's privileges — root, for
// every system package manager — so "../../etc/cron.d/x" would land the write
// outside every declared boundary.
func validateRepositoryName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("repository name is empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid repository name %q: must not contain path separators or %q", name, "..")
	}
	return nil
}

// processRepository runs one repository's provider action steps.
func processRepository(repo types.Repository, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	if err := validateRepositoryName(repo.Name); err != nil {
		return err
	}

	provider, exists := system.GetProvider(repo.PackageManager)
	if !exists {
		return fmt.Errorf("unsupported package manager: %s", repo.PackageManager)
	}

	// A signing key fetched without a declared digest is trusted on nothing
	// but the TLS connection that served it. Warn now; a later major refuses
	// (the policy ratchets, never loosens).
	if repo.Action == "add" && repo.KeyURL != "" && repo.KeySha256 == "" {
		log.Warnf("Repository %s downloads its signing key from %s with no key_sha256 declared — "+
			"the key is unpinned. Add key_sha256 to the repository entry; a future major version will refuse this.",
			repo.Name, repo.KeyURL)
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
	data, err := repositoryStepData(repo, repoConfig.Paths, provider)
	if err != nil {
		return err
	}
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
			dest, err := repositoryWritePath(step.Dest, repoConfig.Paths)
			if err != nil {
				return fmt.Errorf("error downloading for %s: %w", repo.Name, err)
			}
			if system.IsDryRun() {
				log.Infof("[DRY-RUN] Would download file: %s -> %s", step.Source, dest)
				continue
			}
			if err := system.DownloadFileWithChecksum(step.Source, dest, provider.Elevated, step.Sha256); err != nil {
				return fmt.Errorf("error downloading file: %w", err)
			}
			continue
		case "write":
			dest, err := repositoryWritePath(step.Dest, repoConfig.Paths)
			if err != nil {
				return fmt.Errorf("error writing for %s: %w", repo.Name, err)
			}
			if system.IsDryRun() {
				log.Infof("[DRY-RUN] Would write file: %s", dest)
				continue
			}
			if err := system.WriteToFile(dest, step.Content, provider.Elevated); err != nil {
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
			dest, err := repositoryWritePath(step.Dest, repoConfig.Paths)
			if err != nil {
				return fmt.Errorf("error copying for %s: %w", repo.Name, err)
			}
			if system.IsDryRun() {
				log.Infof("[DRY-RUN] Would copy file: %s -> %s", step.Source, dest)
				continue
			}
			if err := system.CopyFile(step.Source, dest, provider.Elevated, osInfo); err != nil {
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

	return nil
}

// runRepositoryUpdates refreshes package lists for the providers this
// blueprint file declared repositories for. It used to update every available
// provider after every repositories file — a tree with three such files ran
// `apt update`, `dnf makecache` etc. three times each, for managers whose
// repositories never changed.
//
// A failed update is recorded in the ledger, not just logged: the operator's
// new repository is not actually usable until its package list refresh
// succeeds, so the run's exit code has to reflect the failure.
func runRepositoryUpdates(initConfig *types.InitConfig, declared map[string]bool) error {
	available := system.GetAvailableProviders()
	for name, provider := range available {
		if !declared[name] || provider.Commands.Update == "" {
			continue
		}

		log.Infof("Processing %s Updates", name)
		updateCmd := types.Command{
			Exec:     provider.BinPath,
			Args:     strings.Fields(provider.Commands.Update),
			Elevated: provider.Elevated,
		}

		if err := system.RunCommand(updateCmd, initConfig.Variables.Flags.Debug); err != nil {
			recordFailure("repositories", name+" update", err)
			continue
		}
	}

	return nil
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
