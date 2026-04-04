package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/processors"

	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run All Blueprints - New System Initialization",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle GitHub OAuth authentication if --gh-auth flag is set
		if ghAuth {
			token, err := processors.AuthenticateWithGitHub(initConfig)
			if err != nil {
				return fmt.Errorf("GitHub authentication failed: %w", err)
			}
			// Update the token in both global var and initConfig
			ghApiToken = token
			initConfig.Variables.Flags.GHAPIToken = token
		}

		log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)
		if err := processors.All(initConfig, osInfo, nil); err != nil {
			return fmt.Errorf("error running all processors: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
