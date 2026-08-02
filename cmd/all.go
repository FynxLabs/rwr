package cmd

import (
	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run All Blueprints - New System Initialization",
	Long:  `Run everything. Same as bare "rwr run".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEverything()
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
