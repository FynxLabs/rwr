package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
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
	Run: func(cmd *cobra.Command, args []string) {
		// Get path from args or use current directory
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		// Resolve absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.With("err", err).Errorf("Error resolving path: %s", path)
			os.Exit(1)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Errorf("Path does not exist: %s", absPath)
			os.Exit(1)
		}

		// If no flags specified, determine type from path
		if !validateBlueprints && !validateProviders {
			fileInfo, err := os.Stat(absPath)
			if err != nil {
				log.With("err", err).Errorf("Error accessing path: %s", absPath)
				os.Exit(1)
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
			log.With("err", err).Errorf("Error during validation")
			os.Exit(1)
		}

		// Display results
		if results.ErrorCount > 0 {
			fmt.Printf("Validation failed with %d errors and %d warnings\n", results.ErrorCount, results.WarningCount)
			os.Exit(1)
		} else if results.WarningCount > 0 {
			fmt.Printf("Validation completed with %d warnings\n", results.WarningCount)
		} else {
			fmt.Println("Validation completed successfully")
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	// Add flags
	validateCmd.Flags().BoolVar(&validateBlueprints, "blueprints", false, "Force validation as blueprint files")
	validateCmd.Flags().BoolVar(&validateProviders, "providers", false, "Force validation as provider configurations")
	validateCmd.Flags().BoolVar(&validateVerbose, "verbose", false, "Show detailed validation information")
}
