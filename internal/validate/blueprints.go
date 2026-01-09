package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// ValidateBlueprints validates blueprint files in the specified directory.
// It searches for an init file, validates its structure, and then validates
// all blueprint files in the directory with matching extensions. Each blueprint
// file is validated based on its type (packages, repositories, files, git, etc.).
// Returns an error if validation encounters a critical failure.
func ValidateBlueprints(path string, verbose bool, results *types.ValidationResults, osInfo *types.OSInfo) error {
	log.Infof("Validating blueprints in %s", path)

	// Find init file in the specified directory only
	initFile := findInitFile(path)
	if initFile == "" {
		AddIssue(results, types.ValidationError, "Failed to find init file", path, 0, "Create an init file in the specified directory")
		return nil // Continue with other validations
	}

	// Validate init file
	initConfig, err := validateInitFile(initFile, results)
	if err != nil {
		return fmt.Errorf("error validating init file: %w", err)
	}

	if initConfig == nil {
		// If init config is nil, we can't continue with blueprint validation
		return nil
	}

	// Only validate files in the specified directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	// Get the init file extension to match other blueprint files
	initExt := filepath.Ext(initFile)

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}

		filePath := filepath.Join(path, entry.Name())
		if filePath == initFile {
			continue // Skip init file
		}

		// Only process files with the same extension as the init file
		if filepath.Ext(filePath) == initExt {
			if err := validateBlueprintFile(filePath, initConfig, results); err != nil {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Error validating blueprint file: %s", err), filePath, 0, "")
			}
		}
	}

	if verbose {
		log.Infof("Blueprint validation completed")
	}

	return nil
}

// findInitFile searches for an init file in the specified directory (non-recursive)
func findInitFile(dir string) string {
	// Check for init files with common extensions
	for _, ext := range []string{".json", ".yaml", ".yml", ".toml"} {
		initFile := filepath.Join(dir, "init"+ext)
		if _, err := os.Stat(initFile); err == nil {
			return initFile
		}
	}
	return ""
}

// validateInitFile validates an init file
func validateInitFile(initFile string, results *types.ValidationResults) (*types.InitConfig, error) {
	log.Debugf("Validating init file: %s", initFile)

	var initConfig types.InitConfig

	// Read the init file
	initData, err := os.ReadFile(initFile)
	if err != nil {
		AddIssue(results, types.ValidationError, fmt.Sprintf("Error reading init file: %s", err), initFile, 0, "")
		return nil, nil
	}

	// Unmarshal the init file data
	err = helpers.UnmarshalBlueprint(initData, filepath.Ext(initFile)[1:], &initConfig)
	if err != nil {
		AddIssue(results, types.ValidationError, fmt.Sprintf("Error unmarshaling init file: %s", err), initFile, 0, "Check file format and syntax")
		return nil, nil
	}

	// Validate the Init field
	if initConfig.Init.Format == "" {
		AddIssue(results, types.ValidationError, "Missing required field 'init.format'", initFile, 0, "Add format field to init section")
	}

	// Location is optional, but if provided, validate it
	if initConfig.Init.Location != "" {
		if _, err := os.Stat(initConfig.Init.Location); os.IsNotExist(err) {
			AddIssue(results, types.ValidationWarning, fmt.Sprintf("Location does not exist: %s", initConfig.Init.Location), initFile, 0, "Create the directory or update the location")
		}
	}

	// Validate the PackageManagers field
	if initConfig.PackageManagers != nil {
		for i, pm := range initConfig.PackageManagers {
			if pm.Name == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'packageManagers[%d].name'", i), initFile, 0, "Add name field to package manager")
			}
			if pm.Action == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'packageManagers[%d].action'", i), initFile, 0, "Add action field to package manager")
			}
		}
	}

	// Validate the Repositories field
	if initConfig.Repositories != nil {
		for i, repo := range initConfig.Repositories {
			if repo.Name == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].name'", i), initFile, 0, "Add name field to repository")
			}
			if repo.PackageManager == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].package_manager'", i), initFile, 0, "Add package_manager field to repository")
			}
			if repo.Action == "" {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Missing required field 'repositories[%d].action'", i), initFile, 0, "Add action field to repository")
			}
		}
	}

	return &initConfig, nil
}

// validateBlueprintFile validates a blueprint file
func validateBlueprintFile(blueprintFile string, initConfig *types.InitConfig, results *types.ValidationResults) error {
	log.Debugf("Validating blueprint file: %s", blueprintFile)

	// Read and process the blueprint file
	blueprintFileData, err := os.ReadFile(blueprintFile)
	if err != nil {
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Resolve template variables if it's a bootstrap file
	if filepath.Base(blueprintFile) == "bootstrap.yaml" {
		blueprintFileData, err = helpers.ResolveTemplate(blueprintFileData, initConfig.Variables)
		if err != nil {
			return fmt.Errorf("error resolving variables in bootstrap file: %w", err)
		}
	}

	// Determine blueprint type from filename or directory name
	filename := filepath.Base(blueprintFile)
	dir := filepath.Base(filepath.Dir(blueprintFile))

	var blueprintType string
	switch {
	case filename == "bootstrap.yaml" || filename == "bootstrap.yml" || filename == "bootstrap.json" || filename == "bootstrap.toml":
		blueprintType = "bootstrap"
	default:
		blueprintType = strings.ToLower(dir)
	}

	if blueprintType == "" {
		AddIssue(results, types.ValidationWarning, fmt.Sprintf("Could not determine blueprint type for: %s", blueprintFile), blueprintFile, 0, "")
		return nil
	}

	log.Debugf("Processing %s from file: %s", blueprintType, blueprintFile)

	// Validate based on blueprint type
	switch blueprintType {
	case "bootstrap":
		var bootstrapData types.BootstrapData
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &bootstrapData)
		if err != nil {
			return fmt.Errorf("error unmarshaling bootstrap blueprint: %w", err)
		}
		ValidateBootstrap(bootstrapData, blueprintFile, results)

	case "packages":
		var packagesData types.PackagesData
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &packagesData)
		if err != nil {
			return fmt.Errorf("error unmarshaling packages blueprint: %w", err)
		}
		ValidatePackages(packagesData.Packages, blueprintFile, results)

	case "repositories":
		var repositories []types.Repository
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &repositories)
		if err != nil {
			return fmt.Errorf("error unmarshaling repositories blueprint: %w", err)
		}
		ValidateRepositories(repositories, blueprintFile, results)

	case "files":
		var files []types.File
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &files)
		if err != nil {
			return fmt.Errorf("error unmarshaling files blueprint: %w", err)
		}
		ValidateFiles(files, blueprintFile, results)

	case "git":
		var gitRepositories []types.Git
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &gitRepositories)
		if err != nil {
			return fmt.Errorf("error unmarshaling git repositories blueprint: %w", err)
		}
		ValidateGitRepositories(gitRepositories, blueprintFile, results)

	case "scripts":
		var scripts []types.Script
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &scripts)
		if err != nil {
			return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
		}
		ValidateScripts(scripts, blueprintFile, results)

	case "services":
		var services []types.Service
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &services)
		if err != nil {
			return fmt.Errorf("error unmarshaling services blueprint: %w", err)
		}
		ValidateServices(services, blueprintFile, results)

	case "ssh_keys":
		var sshKeys []types.SSHKey
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &sshKeys)
		if err != nil {
			return fmt.Errorf("error unmarshaling ssh keys blueprint: %w", err)
		}
		ValidateSSHKeys(sshKeys, blueprintFile, results)

	case "users":
		var usersData types.UsersData
		err = helpers.UnmarshalBlueprint(blueprintFileData, filepath.Ext(blueprintFile)[1:], &usersData)
		if err != nil {
			return fmt.Errorf("error unmarshaling users blueprint: %w", err)
		}
		ValidateUsers(usersData.Users, blueprintFile, results)

	default:
		AddIssue(results, types.ValidationWarning, fmt.Sprintf("Unsupported blueprint type: %s", blueprintType), blueprintFile, 0, "")
	}

	return nil
}
