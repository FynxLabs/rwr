package cmd

import (
	"github.com/spf13/cobra"
)

// newAllCmd is the whole-tree run: new-system initialization.
func newAllCmd(app *AppConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run All Blueprints - New System Initialization",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEverything(app)
		},
	}
}
