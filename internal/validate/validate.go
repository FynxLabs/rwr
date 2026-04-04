// Package validate provides validation functionality for blueprints and providers.
// It validates blueprint structure, component definitions, bootstrap configurations,
// and provider availability. The package reports validation issues with severity
// levels (error, warning, info) and provides detailed feedback for troubleshooting
// blueprint configuration problems.
package validate

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// Validate runs blueprint and provider validation based on the provided options.
// It returns aggregated results with issue counts by severity (error, warning, info).
func Validate(options types.ValidationOptions, osInfo *types.OSInfo) (*types.ValidationResults, error) {
	results := &types.ValidationResults{
		Issues: []types.ValidationIssue{},
	}

	// Validate blueprints if requested
	if options.ValidateBlueprints {
		if err := ValidateBlueprints(options.Path, options.Verbose, results, osInfo); err != nil {
			return results, fmt.Errorf("error validating blueprints: %w", err)
		}
	}

	// Validate providers if requested
	if options.ValidateProviders {
		if err := ValidateProviders(options.Path, options.Verbose, results, osInfo); err != nil {
			return results, fmt.Errorf("error validating providers: %w", err)
		}
	}

	// Count issues by severity
	for _, issue := range results.Issues {
		switch issue.Severity {
		case types.ValidationError:
			results.ErrorCount++
		case types.ValidationWarning:
			results.WarningCount++
		case types.ValidationInfo:
			results.InfoCount++
		}
	}

	return results, nil
}

// AddIssue records a validation issue and logs it at the appropriate severity level.
// If file and line are provided, they are included in the log output for traceability.
func AddIssue(results *types.ValidationResults, severity types.ValidationSeverity, message string, file string, line int, suggestion string) {
	issue := types.ValidationIssue{
		Severity:   severity,
		Message:    message,
		File:       file,
		Line:       line,
		Suggestion: suggestion,
	}

	results.Issues = append(results.Issues, issue)

	// Log the issue
	logMsg := fmt.Sprintf("%s: %s", severity, message)
	if file != "" {
		logMsg += fmt.Sprintf(" [%s", file)
		if line > 0 {
			logMsg += fmt.Sprintf(":%d", line)
		}
		logMsg += "]"
	}
	if suggestion != "" {
		logMsg += fmt.Sprintf(" - Suggestion: %s", suggestion)
	}

	switch severity {
	case types.ValidationError:
		log.Error(logMsg)
	case types.ValidationWarning:
		log.Warn(logMsg)
	case types.ValidationInfo:
		log.Info(logMsg)
	}
}
