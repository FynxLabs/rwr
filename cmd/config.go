package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thefynx/rwr/internal/helpers"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Create or modify rwr configuration",
	Long:  `Create or modify rwr configuration for JumpCloud and rwr settings`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please specify a sub-command. See rwr config --help for more information.")
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize rwr configuration",
	Long:  `Initialize rwr configuration file with default settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := helpers.CreateDefaultConfig()
		if err != nil {
			fmt.Println("Error initializing configuration:", err)
			os.Exit(1)
		}
		fmt.Println("Configuration initialized successfully.")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
}
