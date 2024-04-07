package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors"
	"os"

	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run All Blueprints - New System Initialization",
	Run: func(cmd *cobra.Command, args []string) {
		err := processors.All(systemInfo)
		if err != nil {
			log.With("err", err).Errorf("Error initializing system information")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
