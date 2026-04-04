package validate

import (
	"fmt"
	"path/filepath"
	"strings"

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
