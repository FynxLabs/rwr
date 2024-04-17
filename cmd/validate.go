package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// validateCmd validates the RWR Blueprints
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the RWR Blueprints",
	Run: func(cmd *cobra.Command, args []string) {
		//err := processors.ValidateBlueprints(initConfig)
		//if err != nil {
		//	log.With("err", err).Errorf("Error validating blueprints")
		//	os.Exit(1)
		//}
		//fmt.Println("Blueprints validated successfully")
		log.Warn("Blueprint Validation is not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
