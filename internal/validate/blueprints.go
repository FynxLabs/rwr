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

// findInitFile searches for an init file in the specified directory (non-recursive).
func findInitFile(dir string) string {
	// Check for init files with common extensions
	for _, ext := range []string{types.FormatExtJSON, types.FormatExtYAML, types.FormatExtYAMLAlt, types.FormatExtTOML} {
		initFile := filepath.Join(dir, "init"+ext)
		if _, err := os.Stat(initFile); err == nil {
			return initFile
		}
	}
	return ""
}

// validateInitFile validates an init file.
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

// validateBlueprintFile validates a blueprint file.
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
	switch filename {
	case "bootstrap.yaml", "bootstrap.yml", "bootstrap.json", "bootstrap.toml":
		blueprintType = types.BlueprintTypeBootstrap
	default:
		blueprintType = strings.ToLower(dir)
	}

	if blueprintType == "" {
		AddIssue(results, types.ValidationWarning, fmt.Sprintf("Could not determine blueprint type for: %s", blueprintFile), blueprintFile, 0, "")
		return nil
	}

	log.Debugf("Processing %s from file: %s", blueprintType, blueprintFile)

	validator, ok := blueprintValidators[blueprintType]
	if !ok {
		AddIssue(results, types.ValidationWarning, fmt.Sprintf("Unsupported blueprint type: %s", blueprintType), blueprintFile, 0, "")
		return nil
	}

	return validator(blueprintFileData, filepath.Ext(blueprintFile)[1:], blueprintFile, results)
}

// blueprintValidator unmarshals and validates a single blueprint type.
type blueprintValidator func(data []byte, format string, file string, results *types.ValidationResults) error

// blueprintValidators maps blueprint types to their unmarshal+validate functions.
var blueprintValidators = map[string]blueprintValidator{
	types.BlueprintTypeBootstrap: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.BootstrapData
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling bootstrap blueprint: %w", err)
		}
		ValidateBootstrap(d, file, results)
		return nil
	},
	types.BlueprintTypePackages: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.PackagesData
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling packages blueprint: %w", err)
		}
		ValidatePackages(d.Packages, file, results)
		return nil
	},
	types.BlueprintTypeRepositories: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.Repository
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling repositories blueprint: %w", err)
		}
		ValidateRepositories(d, file, results)
		return nil
	},
	types.BlueprintTypeFiles: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.File
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling files blueprint: %w", err)
		}
		ValidateFiles(d, file, results)
		return nil
	},
	types.BlueprintTypeGit: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.Git
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling git repositories blueprint: %w", err)
		}
		ValidateGitRepositories(d, file, results)
		return nil
	},
	types.BlueprintTypeScripts: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.Script
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling scripts blueprint: %w", err)
		}
		ValidateScripts(d, file, results)
		return nil
	},
	types.BlueprintTypeServices: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.Service
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling services blueprint: %w", err)
		}
		ValidateServices(d, file, results)
		return nil
	},
	types.BlueprintTypeSSHKeys: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d []types.SSHKey
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling ssh keys blueprint: %w", err)
		}
		ValidateSSHKeys(d, file, results)
		return nil
	},
	types.BlueprintTypeUsers: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.UsersData
		if err := helpers.UnmarshalBlueprint(data, format, &d); err != nil {
			return fmt.Errorf("error unmarshaling users blueprint: %w", err)
		}
		ValidateUsers(d.Users, file, results)
		return nil
	},
}
