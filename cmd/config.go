package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/spf13/cobra"
)

var initFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Create or modify rwr configuration",
	Long: `Create the rwr configuration file. With --create, prompts for each
setting and writes the config; without it, this help is shown — there is no
config view or edit mode yet.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initFlag {
			if err := helpers.CreateDefaultConfig(); err != nil {
				return fmt.Errorf("error initializing configuration: %w", err)
			}
			fmt.Println("Configuration initialized successfully.")
		} else {
			if err := cmd.Help(); err != nil {
				return fmt.Errorf("error displaying help: %w", err)
			}
		}
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVarP(&initFlag, "create", "c", false, "Create the configuration file")
	rootCmd.AddCommand(configCmd)
}
