package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/types"
	"github.com/fynxlabs/rwr/internal/validate"
	"github.com/spf13/cobra"
)

var (
	validateBlueprints bool
	validateProviders  bool
	validateVerbose    bool
)

// validateCmd validates the RWR Blueprints and Provider configurations
var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate RWR Blueprints and Provider configurations",
	Long: `Validate RWR Blueprints and Provider configurations to ensure they are correctly structured
and will work as expected when deployed. This command helps identify issues before running
your configurations, saving time and preventing errors.

Examples:
  # Validate a single provider file
  rwr validate providers/paru.toml

  # Validate all providers in a directory
  rwr validate providers/

  # Force validation as blueprints
  rwr validate path/to/dir --blueprints

  # Force validation as providers
  rwr validate path/to/file --providers`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get path from args or use current directory
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		// Resolve absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("error resolving path %s: %w", path, err)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}

		// If no flags specified, determine type from path
		if !validateBlueprints && !validateProviders {
			fileInfo, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("error accessing path %s: %w", absPath, err)
			}

			if fileInfo.IsDir() {
				// For directories, check if it's the providers directory
				if filepath.Base(absPath) == "providers" {
					validateProviders = true
				} else {
					validateBlueprints = true
				}
			} else {
				// For files, check extension
				if filepath.Ext(absPath) == ".toml" {
					validateProviders = true
				} else {
					validateBlueprints = true
				}
			}
		}

		// Set up validation options
		options := types.ValidationOptions{
			Path:               absPath,
			ValidateBlueprints: validateBlueprints,
			ValidateProviders:  validateProviders,
			Verbose:            validateVerbose,
		}

		// Run validation
		results, err := validate.Validate(options, osInfo)
		if err != nil {
			return fmt.Errorf("error during validation: %w", err)
		}

		// Display results
		if results.ErrorCount > 0 {
			return fmt.Errorf("validation failed with %d errors and %d warnings", results.ErrorCount, results.WarningCount)
		} else if results.WarningCount > 0 {
			fmt.Printf("Validation completed with %d warnings\n", results.WarningCount)
		} else {
			fmt.Println("Validation completed successfully")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	// Add flags
	validateCmd.Flags().BoolVar(&validateBlueprints, "blueprints", false, "Force validation as blueprint files")
	validateCmd.Flags().BoolVar(&validateProviders, "providers", false, "Force validation as provider configurations")
	validateCmd.Flags().BoolVar(&validateVerbose, "verbose", false, "Show detailed validation information")
}
