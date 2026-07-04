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
	Long:  `Create or modify rwr configuration for JumpCloud and rwr settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initFlag {
			if err := helpers.CreateDefaultConfig(); err != nil {
				return fmt.Errorf("error initializing configuration: %w", err)
			}
			fmt.Println("Configuration initialized successfully.")
		} else {
			cmd.Help() //nolint:errcheck
		}
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVarP(&initFlag, "create", "c", false, "Create the configuration file")
	rootCmd.AddCommand(configCmd)
}
