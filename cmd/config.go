package cmd

import (
	"fmt"
	"os"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/spf13/cobra"
)

var initFlag bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Create or modify rwr configuration",
	Long:  `Create or modify rwr configuration for JumpCloud and rwr settings`,
	Run: func(cmd *cobra.Command, args []string) {
		if initFlag {
			err := helpers.CreateDefaultConfig()
			if err != nil {
				fmt.Println("Error initializing configuration:", err)
				os.Exit(1)
			}
			fmt.Println("Configuration initialized successfully.")
		} else {
			cmd.Help()
		}
	},
}

func init() {
	configCmd.Flags().BoolVarP(&initFlag, "create", "c", false, "Create the configuration file")
	rootCmd.AddCommand(configCmd)
}
