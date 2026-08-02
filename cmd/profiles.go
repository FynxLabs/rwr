package cmd

import (
	"fmt"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/spf13/cobra"
)

func newProfilesCmd(app *AppConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "Discover and list available profiles in your configuration",
		Long: `The profiles command reads your blueprints and lists every profile they declare.
This tells you what you can pass to --profile.

Profiles let you select what applies to a machine (work, personal, development,
gaming, and so on). An entry that declares no profiles is a base item and always
applies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.InitConfig == nil {
				return fmt.Errorf("configuration not initialized: check that you have a valid init file")
			}

			summary, err := processors.CollectProfiles(app.InitConfig)
			if err != nil {
				return fmt.Errorf("error reading profiles: %w", err)
			}

			log.Debugf("Inspected %d blueprint file(s) in %s", summary.Files, app.InitConfig.Init.Location)

			if len(summary.Names) == 0 {
				fmt.Println("No profiles found in your blueprints.")
				fmt.Printf("All %d item(s) are base items and always apply.\n", summary.BaseItems)
				return nil
			}

			fmt.Printf("Available profiles (%d found):\n\n", len(summary.Names))
			for _, profile := range summary.Names {
				fmt.Printf("  • %s (%d items)\n", profile, summary.Counts[profile])
			}
			fmt.Printf("\n  base items (always applied): %d\n", summary.BaseItems)

			fmt.Println()
			fmt.Println("Usage examples:")
			fmt.Printf("  rwr all --profile %s\n", summary.Names[0])
			if len(summary.Names) > 1 {
				fmt.Printf("  rwr all --profile %s --profile %s\n", summary.Names[0], summary.Names[1])
				fmt.Printf("  rwr run packages --profile %s\n", strings.Join(summary.Names[:2], ","))
			}
			fmt.Println("  rwr all --profile all")
			return nil
		},
	}
}
