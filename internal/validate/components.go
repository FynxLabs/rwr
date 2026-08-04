package validate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
)

// ValidatePackages validates package definitions.
// It checks that each package has required fields (name, action) and validates
// that the action is one of the supported types (install, remove, update).
// It also verifies that specified package managers exist in the system.
// Validation issues are added to the results parameter.
func ValidatePackages(packages []types.Package, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, pkg := range packages {
		if validateImport(pkg.Import, fmt.Sprintf("packages[%d]", i), blueprintDir, file, results, &types.PackagesData{}) {
			continue
		}

		// Only the names list is processed when both are declared; say so
		// rather than silently ignoring one of them.
		if pkg.Name != "" && len(pkg.Names) > 0 {
			AddIssue(results, types.ValidationWarning,
				fmt.Sprintf("packages[%d] declares both 'name' and 'names'; only the names list is processed", i),
				file, 0, "Remove one of the two fields")
		}

		// A package entry names one package with `name` or several with `names`.
		// Requiring `name` and separately warning on an empty `names` reported two
		// problems against every correct entry, whichever form it used.
		if pkg.Name == "" && len(pkg.Names) == 0 {
			AddIssue(results, types.ValidationError,
				fmt.Sprintf("Missing required field 'packages[%d].name'", i), file, 0,
				"Add a name field, or a names list, to the package")
		}

		validateEnum(pkg.Action, fmt.Sprintf("packages[%d].action", i),
			[]string{types.ActionInstall, types.ActionRemove}, file, results)

		// The processor refuses names beginning with '-': argv-exec'd, such a
		// name reads as an option to the elevated package manager. What a
		// processor rejects, validate flags.
		for _, name := range append([]string{pkg.Name}, pkg.Names...) {
			if strings.HasPrefix(name, "-") {
				AddIssue(results, types.ValidationError,
					fmt.Sprintf("packages[%d]: package name %q may not begin with '-'", i, name),
					file, 0, "It would be read as an option by the package manager")
			}
		}

		// package_manager is optional: without one, the package is installed by the
		// default manager detected for this machine, which is the common case.
		if pkg.PackageManager != "" {
			validateProviderExists(pkg.PackageManager, "package", packageLabel(pkg), file, results)
		}
	}
}

// ValidateRepositories validates repository definitions.
// It checks that each repository has required fields (name, package_manager, action)
// and validates that the action is either 'add' or 'remove'. It also verifies
// that specified package managers exist and that add actions have a URL.
// Validation issues are added to the results parameter.
func ValidateRepositories(repositories []types.Repository, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, repo := range repositories {
		if validateImport(repo.Import, fmt.Sprintf("repositories[%d]", i), blueprintDir, file, results, &types.RepositoriesData{}) {
			continue
		}

		validateRequired(repo.Name, fmt.Sprintf("repositories[%d].name", i), file, results, "Add name field to repository")

		validateRequired(repo.PackageManager, fmt.Sprintf("repositories[%d].package_manager", i), file, results, "Add package_manager field to repository")
		validateProviderExists(repo.PackageManager, "repository", repo.Name, file, results)

		validateEnum(repo.Action, fmt.Sprintf("repositories[%d].action", i),
			[]string{types.RepoActionAdd, types.RepoActionRemove}, file, results)

		if repo.URL == "" && repo.Action == types.RepoActionAdd {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No URL specified for repository '%s'", repo.Name), file, 0, "Add URL field to repository")
		}
	}
}

// ValidateFiles validates file definitions.
// It checks that each file has required fields (target, action) and validates
// that the action is one of the supported types (create, delete, append, template).
// It verifies that create/append/template actions have content or source, and
// warns about relative paths. Validation issues are added to the results parameter.
func ValidateFiles(files []types.File, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, f := range files {
		if validateImport(f.Import, fmt.Sprintf("files[%d]", i), blueprintDir, file, results, &types.FileData{}) {
			continue
		}

		validateRequired(f.Target, fmt.Sprintf("files[%d].target", i), file, results, "Add target field to file")

		validateEnum(f.Action, fmt.Sprintf("files[%d].action", i), types.FileActions, file, results)

		if f.Action == types.FileActionCreate || f.Action == types.FileActionCopy || f.Action == types.FileActionMove {
			if f.Content == "" && f.Source == "" {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("No content or source specified for file '%s'", f.Target), file, 0, "Add content or source field to file")
			}
		}

		validateFileMode(f.Mode, f.Action, fmt.Sprintf("files[%d]", i), file, results)

		validatePath(f.Target, fmt.Sprintf("file '%s'", f.Target), file, results)
	}
}

// ValidateDirectories checks directory entries the same way ValidateFiles
// checks files: they share the action vocabulary and the mode rules, and the
// files processor dispatches both — validating only two of the three kinds a
// files blueprint carries let a bad directory mode through to run time.
func ValidateDirectories(directories []types.Directory, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, d := range directories {
		if validateImport(d.Import, fmt.Sprintf("directories[%d]", i), blueprintDir, file, results, &types.FileData{}) {
			continue
		}

		validateRequired(d.Target, fmt.Sprintf("directories[%d].target", i), file, results, "Add target field to directory")

		validateEnum(d.Action, fmt.Sprintf("directories[%d].action", i), types.FileActions, file, results)

		validateFileMode(d.Mode, d.Action, fmt.Sprintf("directories[%d]", i), file, results)

		validatePath(d.Target, fmt.Sprintf("directory '%s'", d.Target), file, results)
	}
}

// modeCarryingActions are the actions that apply a declared mode. Every other
// action ignores it: a symlink has no mode of its own, and delete, move, chown
// and chgrp do not touch it.
var modeCarryingActions = map[string]bool{
	types.FileActionCreate: true,
	types.FileActionChmod:  true,
	types.FileActionCopy:   true,
}

// validateFileMode reports the mode problems that survive decoding.
//
// A mode written ambiguously — `mode: 644`, which as a number is 0o1204 — is
// already refused by types.FileMode while the blueprint is being read, so it
// arrives here as a parse error naming the file. What is left is a mode that
// parses but cannot do what the entry asks: a chmod with nothing to chmod to,
// which would otherwise strip every permission off the target at run time, and
// modes that are applied but hand out more access than a blueprint usually
// means to.
func validateFileMode(mode types.FileMode, action, field, file string, results *types.ValidationResults) {
	if action == types.FileActionChmod && !mode.IsSet() {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Missing required field '%s.mode' for the chmod action", field), file, 0,
			`Add a mode, written as a quoted octal string such as mode: "0644"`)
		return
	}

	if !mode.IsSet() {
		return
	}

	if mode > types.MaxFileMode {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Mode %s in '%s.mode' is not a permission mode", mode, field), file, 0,
			`Use at most four octal digits, such as mode: "0644"`)
		return
	}

	if !modeCarryingActions[action] && action != "" {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Mode %s in '%s.mode' is ignored by the '%s' action", mode, field, action), file, 0,
			"Drop the mode, or use a create or chmod action to apply it")
	}

	if mode&0o002 != 0 {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Mode %s in '%s.mode' is world-writable", mode, field), file, 0,
			`Drop the world-write bit, for example mode: "0644"`)
	}

	if mode&0o6000 != 0 {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Mode %s in '%s.mode' sets setuid or setgid", mode, field), file, 0,
			`Confirm this is intended; a plain permission mode is four digits starting with 0, such as mode: "0644"`)
	}
}

// ValidateGitRepositories validates git repository definitions.
// It checks that each git repository has required fields (url, path) and
// warns about relative paths that should use absolute paths or ~ prefix.
// Validation issues are added to the results parameter.
func ValidateGitRepositories(gitRepositories []types.Git, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, repo := range gitRepositories {
		if validateImport(repo.Import, fmt.Sprintf("git[%d]", i), blueprintDir, file, results, &types.GitData{}) {
			continue
		}

		validateRequired(repo.URL, fmt.Sprintf("git[%d].url", i), file, results, "Add URL field to git repository")
		validateRequired(repo.Path, fmt.Sprintf("git[%d].path", i), file, results, "Add path field to git repository")

		validatePath(repo.Path, fmt.Sprintf("git repository '%s'", repo.URL), file, results)

		if repo.Action != "" {
			validateEnum(repo.Action, fmt.Sprintf("git[%d].action", i),
				[]string{types.GitActionClone, types.GitActionPull}, file, results)
		}
	}
}

// ValidateScripts validates script definitions.
// It checks that each script has a name and either an exec command or content.
// At least one of exec or content must be specified for the script to be valid.
// Validation issues are added to the results parameter.
func ValidateScripts(scripts []types.Script, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, script := range scripts {
		if validateImport(script.Import, fmt.Sprintf("scripts[%d]", i), blueprintDir, file, results, &types.ScriptData{}) {
			continue
		}

		validateRequired(script.Name, fmt.Sprintf("scripts[%d].name", i), file, results, "Add name field to script")

		// A script comes from a file on disk (source), from inline content, or is a
		// program named by exec. The processor accepts all three and defaults exec to
		// the platform shell; validation used to accept only two, so every script
		// declared with `source` was reported as missing a required field.
		if script.Exec == "" && script.Content == "" && script.Source == "" {
			AddIssue(results, types.ValidationError,
				fmt.Sprintf("Missing required field 'scripts[%d].exec', 'scripts[%d].content' or 'scripts[%d].source'", i, i, i),
				file, 0, "Add an exec, content, or source field to the script")
		}
	}
}

// ValidateServices validates service definitions.
// It checks that each service has required fields (name, action) and validates
// that the action is one of the types the services processor implements. It used
// to accept only five of the nine, so a valid `reload` or `status` blueprint was
// reported as an error by `rwr validate` and then ran correctly.
// Validation issues are added to the results parameter.
func ValidateServices(services []types.Service, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, service := range services {
		if validateImport(service.Import, fmt.Sprintf("services[%d]", i), blueprintDir, file, results, &types.ServiceData{}) {
			continue
		}

		validateRequired(service.Name, fmt.Sprintf("services[%d].name", i), file, results, "Add name field to service")

		validateEnum(service.Action, fmt.Sprintf("services[%d].action", i),
			[]string{
				types.ServiceActionEnable, types.ServiceActionDisable,
				types.ServiceActionStart, types.ServiceActionStop, types.ServiceActionRestart,
				types.ServiceActionReload, types.ServiceActionStatus,
				types.ServiceActionCreate, types.ServiceActionDelete,
			}, file, results)
	}
}

// ValidateSSHKeys validates SSH key definitions.
// It checks that each SSH key has a name and validates the key type if specified
// (rsa, ed25519, ecdsa are recommended). It also verifies that paths are absolute
// or use the ~ prefix. Validation issues are added to the results parameter.
func ValidateSSHKeys(sshKeys []types.SSHKey, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, key := range sshKeys {
		if validateImport(key.Import, fmt.Sprintf("ssh_keys[%d]", i), blueprintDir, file, results, &types.SSHKeyData{}) {
			continue
		}

		validateRequired(key.Name, fmt.Sprintf("ssh_keys[%d].name", i), file, results, "Add name field to SSH key")

		if key.Type == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No type specified for SSH key '%s'", key.Name), file, 0, "Add type field to SSH key")
		} else if key.Type != "rsa" && key.Type != "ed25519" && key.Type != "ecdsa" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Unusual type '%s' for SSH key '%s'", key.Type, key.Name), file, 0, "Use 'rsa', 'ed25519', or 'ecdsa'")
		}

		if key.Path == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No path specified for SSH key '%s'", key.Name), file, 0, "Add path field to SSH key")
		} else {
			validatePath(key.Path, fmt.Sprintf("SSH key '%s'", key.Name), file, results)
		}
	}
}

// ValidateUsers validates user definitions.
// It checks that each user has required fields (name, action) and validates
// that the action is one of the supported types (create, modify, delete).
// Validation issues are added to the results parameter.
func ValidateUsers(users []types.User, file string, results *types.ValidationResults) {
	blueprintDir := filepath.Dir(file)
	for i, user := range users {
		if validateImport(user.Import, fmt.Sprintf("users[%d]", i), blueprintDir, file, results, &types.UsersData{}) {
			continue
		}

		validateRequired(user.Name, fmt.Sprintf("users[%d].name", i), file, results, "Add name field to user")

		validateEnum(user.Action, fmt.Sprintf("users[%d].action", i),
			[]string{types.UserActionCreate, types.UserActionModify, types.UserActionRemove, types.UserActionDelete}, file, results)
	}
}

// packageLabel names a package entry for a message, whichever form it uses.
func packageLabel(pkg types.Package) string {
	if pkg.Name != "" {
		return pkg.Name
	}
	if len(pkg.Names) > 0 {
		return strings.Join(pkg.Names, ", ")
	}
	return "(unnamed)"
}
