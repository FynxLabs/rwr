package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/status"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/uninstall"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newUninstallCmd is wiring; planning and execution live in
// internal/uninstall.
func newUninstallCmd(app *AppConfig) *cobra.Command {
	var yes bool

	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Reverse what recorded runs applied - and only that",
		Long: `Remove what the run journal shows was applied: packages via the provider's
remove verb, files and git checkouts hash-guarded (modified content is
skipped and listed), services disabled, fonts deleted from their recorded
directory. Input is the record, never the blueprint tree; with no record the
command refuses. What cannot be reversed (scripts, configuration writes,
users, uploaded SSH keys, repositories) is listed up front.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := viper.GetString("rwr.configdir")
			entries, err := state.Unreversed(configDir)
			if err != nil {
				return err
			}
			items, notReversible, err := uninstall.Plan(entries)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(notReversible) > 0 {
				helpers.Say(out, "Not reversible (recorded, left untouched):"+"\n")
				for _, line := range notReversible {
					helpers.Say(out, "  %s\n", line)
				}
			}
			if len(items) == 0 {
				helpers.Say(out, "Nothing reversible is recorded."+"\n")
				return nil
			}
			helpers.Say(out, "Will reverse %d item(s):\n", len(items))
			for _, item := range items {
				helpers.Say(out, "  %s\n", item.Action)
			}

			if !yes && !app.DryRun {
				helpers.Say(out, "Proceed? [y/N]: ")
				answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n') //nolint:errcheck // an unreadable answer declines
				if !strings.EqualFold(strings.TrimSpace(answer), "y") {
					helpers.Say(out, "Aborted; nothing was changed."+"\n")
					return nil
				}
			}

			journal, err := state.NewWriter(configDir, "", system.IsDryRun())
			if err != nil {
				return err
			}
			failed := uninstall.Execute(out, items, status.NewQuerier(), journal)
			if err := journal.Finalize(); err != nil {
				log.Warnf("finalizing the journal: %v", err)
			}
			if failed > 0 {
				return fmt.Errorf("%d removal(s) failed; failed entries stay unreversed - re-run to retry them", failed)
			}
			return nil
		},
	}

	uninstallCmd.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation prompt")
	return uninstallCmd
}
