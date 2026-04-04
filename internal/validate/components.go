package validate

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/types"
)

// ValidatePackages validates package definitions.
// It checks that each package has required fields (name, action) and validates
// that the action is one of the supported types (install, remove, update).
// It also verifies that specified package managers exist in the system.
// Validation issues are added to the results parameter.
func ValidatePackages(packages []types.Package, file string, results *types.ValidationResults) {
	for i, pkg := range packages {
		validateRequired(pkg.Name, fmt.Sprintf("packages[%d].name", i), file, results, "Add name field to package")

		validateEnum(pkg.Action, fmt.Sprintf("packages[%d].action", i),
			[]string{types.ActionInstall, types.ActionRemove, types.ActionUpdate}, file, results)

		if pkg.PackageManager == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No package manager specified for package '%s'", pkg.Name), file, 0, "Add package_manager field to package")
		} else {
			validateProviderExists(pkg.PackageManager, "package", pkg.Name, file, results)
		}

		if len(pkg.Names) == 0 {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No package names specified for package '%s'", pkg.Name), file, 0, "Add names field to package")
		}
	}
}

// ValidateRepositories validates repository definitions.
// It checks that each repository has required fields (name, package_manager, action)
// and validates that the action is either 'add' or 'remove'. It also verifies
// that specified package managers exist and that add actions have a URL.
// Validation issues are added to the results parameter.
func ValidateRepositories(repositories []types.Repository, file string, results *types.ValidationResults) {
	for i, repo := range repositories {
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
	for i, f := range files {
		validateRequired(f.Target, fmt.Sprintf("files[%d].target", i), file, results, "Add target field to file")

		validateEnum(f.Action, fmt.Sprintf("files[%d].action", i),
			[]string{types.FileActionCreate, types.FileActionDelete, types.FileActionAppend, types.FileActionTemplate}, file, results)

		if f.Action == types.FileActionCreate || f.Action == types.FileActionAppend || f.Action == types.FileActionTemplate {
			if f.Content == "" && f.Source == "" {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("No content or source specified for file '%s'", f.Target), file, 0, "Add content or source field to file")
			}
		}

		validatePath(f.Target, fmt.Sprintf("file '%s'", f.Target), file, results)
	}
}

// ValidateGitRepositories validates git repository definitions.
// It checks that each git repository has required fields (url, path) and
// warns about relative paths that should use absolute paths or ~ prefix.
// Validation issues are added to the results parameter.
func ValidateGitRepositories(gitRepositories []types.Git, file string, results *types.ValidationResults) {
	for i, repo := range gitRepositories {
		validateRequired(repo.URL, fmt.Sprintf("git[%d].url", i), file, results, "Add URL field to git repository")
		validateRequired(repo.Path, fmt.Sprintf("git[%d].path", i), file, results, "Add path field to git repository")

		validatePath(repo.Path, fmt.Sprintf("git repository '%s'", repo.URL), file, results)
	}
}

// ValidateScripts validates script definitions.
// It checks that each script has a name and either an exec command or content.
// At least one of exec or content must be specified for the script to be valid.
// Validation issues are added to the results parameter.
func ValidateScripts(scripts []types.Script, file string, results *types.ValidationResults) {
	for i, script := range scripts {
		validateRequired(script.Name, fmt.Sprintf("scripts[%d].name", i), file, results, "Add name field to script")

		if script.Exec == "" && script.Content == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'scripts[%d].exec' or 'scripts[%d].content'", i, i), file, 0, "Add exec or content field to script")
		}
	}
}

// ValidateServices validates service definitions.
// It checks that each service has required fields (name, action) and validates
// that the action is one of the supported types (enable, disable, start, stop, restart).
// Validation issues are added to the results parameter.
func ValidateServices(services []types.Service, file string, results *types.ValidationResults) {
	for i, service := range services {
		validateRequired(service.Name, fmt.Sprintf("services[%d].name", i), file, results, "Add name field to service")

		validateEnum(service.Action, fmt.Sprintf("services[%d].action", i),
			[]string{types.ServiceActionEnable, types.ServiceActionDisable, types.ServiceActionStart, types.ServiceActionStop, types.ServiceActionRestart}, file, results)
	}
}

// ValidateSSHKeys validates SSH key definitions.
// It checks that each SSH key has a name and validates the key type if specified
// (rsa, ed25519, ecdsa are recommended). It also verifies that paths are absolute
// or use the ~ prefix. Validation issues are added to the results parameter.
func ValidateSSHKeys(sshKeys []types.SSHKey, file string, results *types.ValidationResults) {
	for i, key := range sshKeys {
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
	for i, user := range users {
		validateRequired(user.Name, fmt.Sprintf("users[%d].name", i), file, results, "Add name field to user")

		validateEnum(user.Action, fmt.Sprintf("users[%d].action", i),
			[]string{types.UserActionCreate, types.UserActionModify, types.UserActionDelete}, file, results)
	}
}
