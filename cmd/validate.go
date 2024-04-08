package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors"
	"os"

	"github.com/spf13/cobra"
)

// validateCmd validates the RWR Blueprints
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the RWR Blueprints",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.ValidateBlueprints(systemInfo)
		if err != nil {
			log.With("err", err).Errorf("Error validating blueprints")
			os.Exit(1)
		}
		fmt.Println("Blueprints validated successfully")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
