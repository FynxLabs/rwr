package validate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ValidatePackages validates package definitions.
// It checks that each package has required fields (name, action) and validates
// that the action is one of the supported types (install, remove, update).
// It also verifies that specified package managers exist in the system.
// Validation issues are added to the results parameter.
func ValidatePackages(packages []types.Package, file string, results *types.ValidationResults) {
	for i, pkg := range packages {
		if pkg.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'packages[%d].name'", i), file, 0, "Add name field to package")
		}

		if pkg.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'packages[%d].action'", i), file, 0, "Add action field to package")
		} else if pkg.Action != "install" && pkg.Action != "remove" && pkg.Action != "update" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Invalid action '%s' for package '%s'", pkg.Action, pkg.Name), file, 0, "Use 'install', 'remove', or 'update'")
		}

		if pkg.PackageManager == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No package manager specified for package '%s'", pkg.Name), file, 0, "Add package_manager field to package")
		} else {
			// Check if package manager exists
			_, exists := system.GetProvider(pkg.PackageManager)
			if !exists {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("Package manager '%s' not found for package '%s'", pkg.PackageManager, pkg.Name), file, 0, "Use an available package manager")
			}
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
		if repo.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].name'", i), file, 0, "Add name field to repository")
		}

		if repo.PackageManager == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].package_manager'", i), file, 0, "Add package_manager field to repository")
		} else {
			// Check if package manager exists
			_, exists := system.GetProvider(repo.PackageManager)
			if !exists {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("Package manager '%s' not found for repository '%s'", repo.PackageManager, repo.Name), file, 0, "Use an available package manager")
			}
		}

		if repo.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].action'", i), file, 0, "Add action field to repository")
		} else if repo.Action != "add" && repo.Action != "remove" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Invalid action '%s' for repository '%s'", repo.Action, repo.Name), file, 0, "Use 'add' or 'remove'")
		}

		if repo.URL == "" && repo.Action == "add" {
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
		if f.Target == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'files[%d].target'", i), file, 0, "Add target field to file")
		}

		if f.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'files[%d].action'", i), file, 0, "Add action field to file")
		} else if f.Action != "create" && f.Action != "delete" && f.Action != "append" && f.Action != "template" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Invalid action '%s' for file '%s'", f.Action, f.Target), file, 0, "Use 'create', 'delete', 'append', or 'template'")
		}

		if f.Action == "create" || f.Action == "append" || f.Action == "template" {
			if f.Content == "" && f.Source == "" {
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("No content or source specified for file '%s'", f.Target), file, 0, "Add content or source field to file")
			}
		}

		// Check if path is absolute or relative
		if !filepath.IsAbs(f.Target) && !strings.HasPrefix(f.Target, "~") {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Relative path specified for file '%s'", f.Target), file, 0, "Use absolute path or path with ~ prefix")
		}
	}
}

// ValidateGitRepositories validates git repository definitions.
// It checks that each git repository has required fields (url, path) and
// warns about relative paths that should use absolute paths or ~ prefix.
// Validation issues are added to the results parameter.
func ValidateGitRepositories(gitRepositories []types.Git, file string, results *types.ValidationResults) {
	for i, repo := range gitRepositories {
		if repo.URL == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'git[%d].url'", i), file, 0, "Add URL field to git repository")
		}

		if repo.Path == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'git[%d].path'", i), file, 0, "Add path field to git repository")
		}

		// Check if path is absolute or relative
		if !filepath.IsAbs(repo.Path) && !strings.HasPrefix(repo.Path, "~") {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Relative path specified for git repository '%s'", repo.URL), file, 0, "Use absolute path or path with ~ prefix")
		}
	}
}

// ValidateScripts validates script definitions.
// It checks that each script has a name and either an exec command or content.
// At least one of exec or content must be specified for the script to be valid.
// Validation issues are added to the results parameter.
func ValidateScripts(scripts []types.Script, file string, results *types.ValidationResults) {
	for i, script := range scripts {
		if script.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'scripts[%d].name'", i), file, 0, "Add name field to script")
		}

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
		if service.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'services[%d].name'", i), file, 0, "Add name field to service")
		}

		if service.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'services[%d].action'", i), file, 0, "Add action field to service")
		} else if service.Action != "enable" && service.Action != "disable" && service.Action != "start" && service.Action != "stop" && service.Action != "restart" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Invalid action '%s' for service '%s'", service.Action, service.Name), file, 0, "Use 'enable', 'disable', 'start', 'stop', or 'restart'")
		}
	}
}

// ValidateSSHKeys validates SSH key definitions.
// It checks that each SSH key has a name and validates the key type if specified
// (rsa, ed25519, ecdsa are recommended). It also verifies that paths are absolute
// or use the ~ prefix. Validation issues are added to the results parameter.
func ValidateSSHKeys(sshKeys []types.SSHKey, file string, results *types.ValidationResults) {
	for i, key := range sshKeys {
		if key.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'ssh_keys[%d].name'", i), file, 0, "Add name field to SSH key")
		}

		if key.Type == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No type specified for SSH key '%s'", key.Name), file, 0, "Add type field to SSH key")
		} else if key.Type != "rsa" && key.Type != "ed25519" && key.Type != "ecdsa" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Unusual type '%s' for SSH key '%s'", key.Type, key.Name), file, 0, "Use 'rsa', 'ed25519', or 'ecdsa'")
		}

		if key.Path == "" {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("No path specified for SSH key '%s'", key.Name), file, 0, "Add path field to SSH key")
		} else if !filepath.IsAbs(key.Path) && !strings.HasPrefix(key.Path, "~") {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Relative path specified for SSH key '%s'", key.Name), file, 0, "Use absolute path or path with ~ prefix")
		}
	}
}

// ValidateUsers validates user definitions.
// It checks that each user has required fields (name, action) and validates
// that the action is one of the supported types (create, modify, delete).
// Validation issues are added to the results parameter.
func ValidateUsers(users []types.User, file string, results *types.ValidationResults) {
	for i, user := range users {
		if user.Name == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'users[%d].name'", i), file, 0, "Add name field to user")
		}

		if user.Action == "" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'users[%d].action'", i), file, 0, "Add action field to user")
		} else if user.Action != "create" && user.Action != "modify" && user.Action != "delete" {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Invalid action '%s' for user '%s'", user.Action, user.Name), file, 0, "Use 'create', 'modify', or 'delete'")
		}
	}
}
