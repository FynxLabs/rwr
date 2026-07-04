package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// validateRequired checks if a required string field is empty and adds a validation error if so.
func validateRequired(value string, fieldPath string, file string, results *types.ValidationResults, suggestion string) {
	if value == "" {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Missing required field '%s'", fieldPath),
			file, 0, suggestion)
	}
}

// validateEnum checks if a value is one of the allowed values.
// If the value is empty, it reports a missing required field error.
// If the value is non-empty but not in the list, it reports an invalid value error.
func validateEnum(value string, fieldPath string, allowedValues []string, file string, results *types.ValidationResults) {
	if value == "" {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Missing required field '%s'", fieldPath),
			file, 0, fmt.Sprintf("Add %s field", fieldPath))
		return
	}

	for _, allowed := range allowedValues {
		if value == allowed {
			return
		}
	}

	AddIssue(results, types.ValidationError,
		fmt.Sprintf("Invalid value '%s' for field '%s'", value, fieldPath),
		file, 0,
		fmt.Sprintf("Use one of: %s", strings.Join(allowedValues, ", ")))
}

// validatePath checks if a path is absolute or uses the ~ prefix.
// Relative paths generate a warning suggesting the use of absolute paths.
func validatePath(path string, fieldName string, file string, results *types.ValidationResults) {
	if path == "" {
		return
	}
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, "~") {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Relative path specified for %s: '%s'", fieldName, path),
			file, 0, "Use absolute path or path with ~ prefix")
	}
}

// validateImport checks that an import path references a valid, parseable blueprint file.
// It verifies the file exists, detects circular imports, and can be unmarshaled as the
// expected blueprint type. Returns true if this item is an import (so callers can skip
// other field validation).
func validateImport(importPath string, fieldPath string, blueprintDir string, file string, results *types.ValidationResults, target interface{}) bool {
	return validateImportWithVisited(importPath, fieldPath, blueprintDir, file, results, target, nil)
}

// validateImportWithVisited performs import validation with circular import detection.
// The visited map tracks already-seen absolute paths to detect cycles.
func validateImportWithVisited(importPath string, fieldPath string, blueprintDir string, file string, results *types.ValidationResults, target interface{}, visited map[string]bool) bool {
	if importPath == "" {
		return false
	}

	fullPath := filepath.Join(blueprintDir, importPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Invalid import path '%s' for %s: %v", importPath, fieldPath, err),
			file, 0, "Use a valid relative path for the import")
		return true
	}

	// Initialize visited map if needed
	if visited == nil {
		visited = make(map[string]bool)
	}

	// Detect circular imports
	if visited[absPath] {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Circular import detected: '%s' for %s", importPath, fieldPath),
			file, 0, "Remove circular import reference")
		return true
	}
	visited[absPath] = true

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Import file not found '%s' for %s", importPath, fieldPath),
			file, 0, "Ensure the import file exists at the specified path")
		return true
	}

	// Try to parse the imported file
	importData, err := os.ReadFile(absPath) //nolint:gosec
	if err != nil {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Cannot read import file '%s' for %s: %v", importPath, fieldPath, err),
			file, 0, "Check file permissions")
		return true
	}

	fileFormat := strings.TrimPrefix(filepath.Ext(absPath), ".")
	if fileFormat == "" {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Import file '%s' has no extension, cannot determine format", importPath),
			file, 0, "Use a file with .yaml, .json, or .toml extension")
		return true
	}

	if target != nil {
		if err := helpers.UnmarshalBlueprint(importData, fileFormat, target); err != nil {
			AddIssue(results, types.ValidationError,
				fmt.Sprintf("Cannot parse import file '%s' for %s: %v", importPath, fieldPath, err),
				file, 0, "Check the import file format and structure")
			return true
		}

		// Recursively validate imported content
		importDir := filepath.Dir(absPath)
		validateImportedContent(target, absPath, importDir, results, visited)
	}

	return true
}

// validateImportedContent recursively validates the items found in an imported blueprint file.
// It dispatches to the appropriate validator based on the target type, using the visited
// map to track circular imports across the entire validation chain.
func validateImportedContent(target interface{}, importFile string, importDir string, results *types.ValidationResults, visited map[string]bool) {
	switch data := target.(type) {
	case *types.PackagesData:
		validatePackagesWithVisited(data.Packages, importFile, results, visited)
	case *types.RepositoriesData:
		validateRepositoriesWithVisited(data.Repositories, importFile, results, visited)
	case *types.FileData:
		validateFilesWithVisited(data.Files, importFile, results, visited)
	case *types.GitData:
		validateGitWithVisited(data.Repos, importFile, results, visited)
	case *types.ScriptData:
		validateScriptsWithVisited(data.Scripts, importFile, results, visited)
	case *types.ServiceData:
		validateServicesWithVisited(data.Services, importFile, results, visited)
	case *types.SSHKeyData:
		validateSSHKeysWithVisited(data.SSHKeys, importFile, results, visited)
	case *types.UsersData:
		validateUsersWithVisited(data.Users, importFile, results, visited)
	}
}

// The following *WithVisited functions validate items and recursively follow imports
// using a shared visited map for circular detection.

func validatePackagesWithVisited(packages []types.Package, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, pkg := range packages {
		if pkg.Import != "" {
			validateImportWithVisited(pkg.Import, fmt.Sprintf("packages[%d]", i), blueprintDir, file, results, &types.PackagesData{}, visited)
			continue
		}
		validateRequired(pkg.Name, fmt.Sprintf("packages[%d].name", i), file, results, "Add name field to package")
		validateEnum(pkg.Action, fmt.Sprintf("packages[%d].action", i),
			[]string{types.ActionInstall, types.ActionRemove, types.ActionUpdate}, file, results)
	}
}

func validateRepositoriesWithVisited(repos []types.Repository, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, repo := range repos {
		if repo.Import != "" {
			validateImportWithVisited(repo.Import, fmt.Sprintf("repositories[%d]", i), blueprintDir, file, results, &types.RepositoriesData{}, visited)
			continue
		}
		validateRequired(repo.Name, fmt.Sprintf("repositories[%d].name", i), file, results, "Add name field to repository")
	}
}

func validateFilesWithVisited(files []types.File, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, f := range files {
		if f.Import != "" {
			validateImportWithVisited(f.Import, fmt.Sprintf("files[%d]", i), blueprintDir, file, results, &types.FileData{}, visited)
			continue
		}
		validateRequired(f.Target, fmt.Sprintf("files[%d].target", i), file, results, "Add target field to file")
	}
}

func validateGitWithVisited(repos []types.Git, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, repo := range repos {
		if repo.Import != "" {
			validateImportWithVisited(repo.Import, fmt.Sprintf("git[%d]", i), blueprintDir, file, results, &types.GitData{}, visited)
			continue
		}
		validateRequired(repo.URL, fmt.Sprintf("git[%d].url", i), file, results, "Add URL field to git repository")
	}
}

func validateScriptsWithVisited(scripts []types.Script, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, script := range scripts {
		if script.Import != "" {
			validateImportWithVisited(script.Import, fmt.Sprintf("scripts[%d]", i), blueprintDir, file, results, &types.ScriptData{}, visited)
			continue
		}
		validateRequired(script.Name, fmt.Sprintf("scripts[%d].name", i), file, results, "Add name field to script")
	}
}

func validateServicesWithVisited(services []types.Service, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, svc := range services {
		if svc.Import != "" {
			validateImportWithVisited(svc.Import, fmt.Sprintf("services[%d]", i), blueprintDir, file, results, &types.ServiceData{}, visited)
			continue
		}
		validateRequired(svc.Name, fmt.Sprintf("services[%d].name", i), file, results, "Add name field to service")
	}
}

func validateSSHKeysWithVisited(keys []types.SSHKey, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, key := range keys {
		if key.Import != "" {
			validateImportWithVisited(key.Import, fmt.Sprintf("ssh_keys[%d]", i), blueprintDir, file, results, &types.SSHKeyData{}, visited)
			continue
		}
		validateRequired(key.Name, fmt.Sprintf("ssh_keys[%d].name", i), file, results, "Add name field to SSH key")
	}
}

func validateUsersWithVisited(users []types.User, file string, results *types.ValidationResults, visited map[string]bool) {
	blueprintDir := filepath.Dir(file)
	for i, user := range users {
		if user.Import != "" {
			validateImportWithVisited(user.Import, fmt.Sprintf("users[%d]", i), blueprintDir, file, results, &types.UsersData{}, visited)
			continue
		}
		validateRequired(user.Name, fmt.Sprintf("users[%d].name", i), file, results, "Add name field to user")
	}
}

// validateProviderExists checks if a named package manager provider exists.
func validateProviderExists(providerName string, itemType string, itemName string, file string, results *types.ValidationResults) {
	if providerName == "" {
		return
	}
	_, exists := system.GetProvider(providerName)
	if !exists {
		AddIssue(results, types.ValidationWarning,
			fmt.Sprintf("Package manager '%s' not found for %s '%s'", providerName, itemType, itemName),
			file, 0, "Use an available package manager")
	}
}
