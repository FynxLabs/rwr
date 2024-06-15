package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/processors"
	"os"

	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run All Blueprints - New System Initialization",
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)
		err := processors.All(initConfig, osInfo, nil)
		if err != nil {
			log.With("err", err).Errorf("Error initializing system information")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
