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
// It verifies the file exists and can be unmarshaled as the expected blueprint type.
// Returns true if this item is an import (so callers can skip other field validation).
func validateImport(importPath string, fieldPath string, blueprintDir string, file string, results *types.ValidationResults, target interface{}) bool {
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

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		AddIssue(results, types.ValidationError,
			fmt.Sprintf("Import file not found '%s' for %s", importPath, fieldPath),
			file, 0, "Ensure the import file exists at the specified path")
		return true
	}

	// Try to parse the imported file
	importData, err := os.ReadFile(absPath)
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
		}
	}

	return true
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
