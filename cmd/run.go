package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thefynx/rwr/internal/processors"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run individual processors",
}

var runPackageCmd = &cobra.Command{
	Use:   "package",
	Short: "Run package processor",
	Run: func(cmd *cobra.Command, args []string) {
		err = processors.ProcessPackages(initConfig.Packages, osInfo)
		if err != nil {
			return nil, fmt.Errorf("error processing packages: %w", err)
		}
	},
}

var runServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Run services processor",
	Run: func(cmd *cobra.Command, args []string) {
		// Run services processor
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runPackageCmd)
	runCmd.AddCommand(runServicesCmd)
}
