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
		// Handle GitHub OAuth authentication if --gh-auth flag is set
		if ghAuth {
			token, err := processors.AuthenticateWithGitHub(initConfig)
			if err != nil {
				log.With("err", err).Errorf("GitHub authentication failed")
				os.Exit(1)
			}
			// Update the token in both global var and initConfig
			ghApiToken = token
			initConfig.Variables.Flags.GHAPIToken = token
		}

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
