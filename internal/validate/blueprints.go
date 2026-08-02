package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/processors"
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

	// Stage-1 resolve produces the routed, template-resolved tree once;
	// validate consumes it instead of walking and rendering a second copy of
	// the same logic (add-tui task 1). The resolver reports per-file problems
	// as diagnostics, so one bad file does not hide the rest of the tree.
	// The tree being validated is the path argument, always: the init file's
	// own location ("." is common) is relative to a run's working directory,
	// not to wherever validate happens to be invoked from.
	initConfig.Init.Location = path
	plan, err := processors.ResolveStage1(initConfig)
	if err != nil {
		return fmt.Errorf("error resolving blueprint tree: %w", err)
	}

	for _, diag := range plan.Diags {
		severity := types.ValidationError
		if diag.Severity == processors.SeverityWarning {
			severity = types.ValidationWarning
		}
		AddIssue(results, severity, diag.Msg, diag.File, diag.Line, "")
	}

	for processor, files := range plan.Files {
		validator, known := blueprintValidators[processor]
		for _, file := range files {
			if !known {
				// The run loop warns about these too; validate says it as well
				// so the operator hears it before a run.
				AddIssue(results, types.ValidationWarning, fmt.Sprintf("Unsupported blueprint type: %s", processor), file.Path, 0, "")
				continue
			}
			if err := validator(file.Resolved, file.Format, file.Path, results); err != nil {
				AddIssue(results, types.ValidationError, fmt.Sprintf("Error validating blueprint file: %s", err), file.Path, 0, "")
			}
		}
	}

	// Bootstrap is dispatched separately from the processor buckets (it is
	// not a run-order blueprint), so validate covers it explicitly.
	for _, name := range helpers.CandidateFilenames("bootstrap") {
		bootstrapFile := filepath.Join(path, name)
		if _, statErr := os.Stat(bootstrapFile); statErr != nil {
			continue
		}
		if err := validateBlueprintFile(bootstrapFile, initConfig, results); err != nil {
			AddIssue(results, types.ValidationError, fmt.Sprintf("Error validating blueprint file: %s", err), bootstrapFile, 0, "")
		}
	}

	if verbose {
		log.Infof("Blueprint validation completed")
	}

	return nil
}

// isBootstrapFileName reports whether a filename is a bootstrap file, in any
// registered format.
func isBootstrapFileName(name string) bool {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	return base == "bootstrap" && helpers.IsBlueprintFile(name)
}

// findInitFile searches for an init file in the specified directory (non-recursive).
func findInitFile(dir string) string {
	for _, name := range helpers.CandidateFilenames("init") {
		initFile := filepath.Join(dir, name)
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
	initData, err := os.ReadFile(initFile) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		AddIssue(results, types.ValidationError, fmt.Sprintf("Error reading init file: %s", err), initFile, 0, "")
		return nil, nil
	}

	// Unmarshal the init file data
	initFormat, formatErr := helpers.FormatForPath(initFile)
	if formatErr != nil {
		AddIssue(results, types.ValidationError, formatErr.Error(), initFile, 0, "Use a file with a supported blueprint extension")
		return nil, nil
	}
	err = helpers.UnmarshalBlueprint(initData, initFormat, &initConfig)
	if err != nil {
		AddIssue(results, types.ValidationError, fmt.Sprintf("Error unmarshaling init file: %s", err), initFile, 0, "Check file format and syntax")
		return nil, nil
	}

	// Blueprints are rendered as templates before they are read, so validation
	// has to render them against the same variables a run would — including
	// the userDefined block the init file declares. Assigning the whole
	// struct used to throw that block away, which blanked every
	// {{ .UserDefined.x }} and turned the values behind them into false
	// "missing required field" errors.
	declared := initConfig.Variables.UserDefined
	if variables, err := helpers.DefaultVariables(); err != nil {
		log.Warnf("Could not resolve template variables for validation: %v", err)
	} else {
		initConfig.Variables = variables
	}
	if initConfig.Variables.UserDefined == nil {
		initConfig.Variables.UserDefined = map[string]interface{}{}
	}
	for key, value := range declared {
		initConfig.Variables.UserDefined[key] = value
	}

	// A tree-wide schema version has to be readable for every blueprint type.
	if err := types.ValidateTreeSchemaVersion(initConfig.Init.SchemaVersion); err != nil {
		AddIssue(results, types.ValidationError, err.Error(), initFile, 0,
			"Declare the version per blueprint file instead, or upgrade rwr")
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

	return &initConfig, nil
}

// validateBlueprintFile validates a blueprint file.
func validateBlueprintFile(blueprintFile string, initConfig *types.InitConfig, results *types.ValidationResults) error {
	log.Debugf("Validating blueprint file: %s", blueprintFile)

	// Read and process the blueprint file
	blueprintFileData, err := os.ReadFile(blueprintFile) // #nosec G304 -- path is operator-supplied blueprint/config input; containment added in PR8
	if err != nil {
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Resolve template variables in every blueprint, as a real run does.
	//
	// This used to apply to bootstrap.yaml alone, which was survivable only because
	// validation never looked past the top directory. A blueprint using
	// {{ .User.home }} is not valid YAML until it is rendered — the braces read as a
	// flow mapping — so validating the raw bytes reports a parse error against a
	// blueprint that works.
	// Strict for the fixed namespaces, lenient only for UserDefined: validate
	// used to render everything with missingkey=zero, so a typo like
	// {{ .User.hoem }} validated clean and failed at run time.
	for _, ref := range helpers.UnknownTemplateReferences(blueprintFileData, initConfig.Variables) {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Template references %s, which does not exist", ref),
			blueprintFile, 0, "Fix the reference; User/System/Flags keys are fixed (UserDefined values are not checked)")
	}

	blueprintFileData, err = helpers.ResolveTemplateForValidation(blueprintFileData, initConfig.Variables)
	if err != nil {
		return fmt.Errorf("error resolving variables in %s: %w", filepath.Base(blueprintFile), err)
	}

	// Determine blueprint type from filename or directory name
	filename := filepath.Base(blueprintFile)
	dir := filepath.Base(filepath.Dir(blueprintFile))

	var blueprintType string
	if isBootstrapFileName(filename) {
		blueprintType = types.BlueprintTypeBootstrap
	} else {
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

	format, formatErr := helpers.FormatForPath(blueprintFile)
	if formatErr != nil {
		AddIssue(results, types.ValidationError, formatErr.Error(), blueprintFile, 0, "Use a file with a supported blueprint extension")
		return nil
	}
	return validator(blueprintFileData, format, blueprintFile, results)
}

// blueprintValidator unmarshals and validates a single blueprint type.
type blueprintValidator func(data []byte, format string, file string, results *types.ValidationResults) error

// blueprintValidators maps blueprint types to their decode+validate functions.
//
// Each one decodes into the same Data struct the matching processor uses. They
// used to decode into a bare slice — []types.Repository for a file whose content
// is `repositories:` followed by a list — which cannot succeed against any real
// blueprint. Nothing caught it because validation never looked inside a
// subdirectory, and blueprints live in subdirectories.
var blueprintValidators = map[string]blueprintValidator{
	types.BlueprintTypeBootstrap: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.BootstrapData
		if err := decode(data, format, types.BlueprintTypeBootstrap, &d); err != nil {
			return err
		}
		ValidateBootstrap(d, file, results)
		return nil
	},
	types.BlueprintTypePackages: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.PackagesData
		if err := decode(data, format, types.BlueprintTypePackages, &d); err != nil {
			return err
		}
		ValidatePackages(d.Packages, file, results)
		return nil
	},
	types.BlueprintTypeRepositories: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.RepositoriesData
		if err := decode(data, format, types.BlueprintTypeRepositories, &d); err != nil {
			return err
		}
		ValidateRepositories(d.Repositories, file, results)
		return nil
	},
	types.BlueprintTypeFiles: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.FileData
		if err := decode(data, format, types.BlueprintTypeFiles, &d); err != nil {
			return err
		}
		// A files blueprint carries files, templates and directories together; the
		// files processor reads all three, so validation has to as well.
		ValidateFiles(append(append([]types.File{}, d.Files...), d.Templates...), file, results)
		ValidateDirectories(d.Directories, file, results)
		return nil
	},
	types.BlueprintTypeGit: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.GitData
		if err := decode(data, format, types.BlueprintTypeGit, &d); err != nil {
			return err
		}
		ValidateGitRepositories(d.Repos, file, results)
		return nil
	},
	types.BlueprintTypeScripts: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.ScriptData
		if err := decode(data, format, types.BlueprintTypeScripts, &d); err != nil {
			return err
		}
		ValidateScripts(d.Scripts, file, results)
		return nil
	},
	types.BlueprintTypeServices: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.ServiceData
		if err := decode(data, format, types.BlueprintTypeServices, &d); err != nil {
			return err
		}
		ValidateServices(d.Services, file, results)
		return nil
	},
	types.BlueprintTypeSSHKeys: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.SSHKeyData
		if err := decode(data, format, types.BlueprintTypeSSHKeys, &d); err != nil {
			return err
		}
		ValidateSSHKeys(d.SSHKeys, file, results)
		return nil
	},
	types.BlueprintTypeUsers: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.UsersData
		if err := decode(data, format, types.BlueprintTypeUsers, &d); err != nil {
			return err
		}
		ValidateUsers(d.Users, file, results)
		return nil
	},
	// fonts and configuration were absent, so every fonts and configuration
	// blueprint was reported as an unsupported type. They decode like the rest;
	// the schema check alone is worth running against them.
	types.BlueprintTypeFonts: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.FontsData
		return decode(data, format, types.BlueprintTypeFonts, &d)
	},
	types.BlueprintTypeConfiguration: func(data []byte, format string, file string, results *types.ValidationResults) error {
		var d types.ConfigData
		return decode(data, format, types.BlueprintTypeConfiguration, &d)
	},
}

// decode reads a blueprint the way a run does, so validation enforces the same
// schema version the processors do.
func decode[T any](data []byte, format, blueprintType string, out *T) error {
	if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, out); err != nil {
		return fmt.Errorf("error unmarshaling %s blueprint: %w", blueprintType, err)
	}
	return nil
}
