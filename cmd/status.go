package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/status"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newStatusCmd is wiring; the drift computation lives in internal/status.
func newStatusCmd(app *AppConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show desired-vs-actual drift without applying anything",
		Long: `Compare the blueprint tree against the machine: what is in sync, missing,
modified since the recorded apply, unknown (not queryable), or stale
(recorded by a past run but no longer in the tree). Read-only: status never
mutates and never elevates. Exits 1 on drift.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := processors.ResolveStage1(app.InitConfig)
			if err != nil {
				return err
			}
			processors.ResolveStage2(plan, app.OSInfo)
			applies, err := state.Applies(viper.GetString("rwr.configdir"))
			if err != nil {
				return err
			}
			rows := status.Rows(plan, applies, status.NewQuerier())
			helpers.Say(cmd.OutOrStdout(), "%s", status.Render(rows, len(applies) > 0))
			if status.Drifted(rows) {
				return fmt.Errorf("drift detected")
			}
			return nil
		},
	}
}
