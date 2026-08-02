package cmd

import (
	"github.com/spf13/cobra"
)

// newAllCmd keeps `rwr all` working for existing scripts, hidden and
// deprecated: `rwr run` is the landing command, and two visible spellings of
// the same run is exactly the duplication the prime namespace should not
// carry.
func newAllCmd(app *AppConfig) *cobra.Command {
	allCmd := &cobra.Command{
		Use:        "all",
		Short:      "Deprecated alias for rwr run",
		Hidden:     true,
		Deprecated: `use "rwr run"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEverything(app)
		},
	}
	return allCmd
}
