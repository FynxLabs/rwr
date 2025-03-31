package types

// ValidationSeverity represents the severity level of a validation issue
type ValidationSeverity string

const (
	// ValidationError represents a critical issue that prevents execution
	ValidationError ValidationSeverity = "ERROR"
	// ValidationWarning represents a non-critical issue that might cause problems
	ValidationWarning ValidationSeverity = "WARNING"
	// ValidationInfo represents informational messages
	ValidationInfo ValidationSeverity = "INFO"
)

// ValidationIssue represents a single validation issue
type ValidationIssue struct {
	Severity    ValidationSeverity
	Message     string
	File        string
	Line        int
	Suggestion  string
	RawMessage  string
	Category    string
	Description string
}

// ValidationResults contains the results of a validation run
type ValidationResults struct {
	Issues       []ValidationIssue
	ErrorCount   int
	WarningCount int
	InfoCount    int
}

// ValidationOptions contains options for the validation process
type ValidationOptions struct {
	Path               string
	ValidateBlueprints bool
	ValidateProviders  bool
	Verbose            bool
}
