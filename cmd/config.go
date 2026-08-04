package cmd

import (
	"fmt"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	var initFlag bool

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "View, edit, or create the rwr configuration",
		Long: `View, edit, or create the rwr configuration file.

"rwr config view" prints the effective merged configuration (secrets
redacted), "rwr config edit" opens it in $VISUAL/$EDITOR, and
"rwr config create" prompts for each setting and writes the file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if initFlag {
				return createConfig()
			}
			return cmd.Help()
		},
	}

	var showSecrets bool
	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "Print the effective merged configuration (secrets redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rendered, err := helpers.ConfigView(showSecrets)
			if err != nil {
				return err
			}
			helpers.Say(cmd.OutOrStdout(), "%s", rendered)
			return nil
		},
	}
	viewCmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Show credential values instead of the redaction placeholder")

	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Open the configuration file in $VISUAL/$EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			return helpers.EditConfig()
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create the configuration file, prompting for each setting",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createConfig()
		},
	}

	configCmd.AddCommand(viewCmd, editCmd, createCmd)

	configCmd.Flags().BoolVarP(&initFlag, "create", "c", false, "Create the configuration file")
	if err := configCmd.Flags().MarkDeprecated("create", "use \"rwr config create\" instead"); err != nil {
		log.Fatalf("marking --create deprecated: %v", err)
	}
	return configCmd
}

func createConfig() error {
	if err := helpers.CreateDefaultConfig(); err != nil {
		return fmt.Errorf("error initializing configuration: %w", err)
	}
	fmt.Println("Configuration initialized successfully.")
	return nil
}
