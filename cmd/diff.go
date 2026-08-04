package cmd

import (
	"fmt"
	"os"

	"github.com/fynxlabs/rwr/internal/diff"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// newDiffCmd is wiring; comparison and routing live in internal/diff.
func newDiffCmd(app *AppConfig) *cobra.Command {
	var (
		format     string
		into       string
		categories = map[string]*bool{}
	)

	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Show machine drift as blueprint material",
		Long: `Compare what is actually on this machine — explicitly-installed packages,
enabled services, git checkouts, configs — against the blueprint tree.
Additions are things you did by hand; removals are declared things that are
gone. Output as a readable list, paste-ready blueprint blocks (--format), or
routed interactively into the tree (--into). Diff never touches the system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			machine := scanMachine(scoped(categories))

			plan, err := processors.ResolveStage1(app.InitConfig)
			if err != nil {
				return err
			}
			processors.ResolveStage2(plan)
			records, err := state.LoadAll(viper.GetString("rwr.configdir"))
			if err != nil {
				return err
			}

			changes := diff.Compute(machine, plan, records)
			changes = filterCategories(changes, scoped(categories))

			switch {
			case into != "":
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("--into needs a terminal for the destination picks; use --format for the non-interactive path")
				}
				return routeInteractively(cmd, changes, into)
			case format != "":
				canonical, err := helpers.CanonicalFormat(format)
				if err != nil {
					return err
				}
				blocks, err := diff.EmitBlocks(changes, machine, canonical)
				if err != nil {
					return err
				}
				helpers.Say(cmd.OutOrStdout(), "%s", blocks)
				return nil
			default:
				helpers.Say(cmd.OutOrStdout(), "%s", diff.Render(changes))
				return nil
			}
		},
	}

	diffCmd.Flags().StringVar(&format, "format", "", "Emit additions as paste-ready blueprint blocks: yaml, json, toml, or cue")
	diffCmd.Flags().StringVar(&into, "into", "", "Route additions into this blueprint tree interactively")
	for _, category := range []string{"packages", "configs", "services", "git"} {
		categories[category] = diffCmd.Flags().Bool(category, false, "Only the "+category+" category")
	}
	return diffCmd
}

// scanMachine gathers the scanned side, scoped to the asked-for categories.
func scanMachine(only map[string]bool) diff.Machine {
	home, _ := os.UserHomeDir() //nolint:errcheck // empty home just disables home-templating
	machine := diff.Machine{Home: home}
	all := len(only) == 0
	if all || only["packages"] {
		machine.Packages = scan.Packages(system.GetAvailableProviders())
	}
	if all || only["services"] {
		machine.Services = scan.Services()
	}
	if all || only["git"] {
		machine.Git = scan.GitCheckouts([]string{home + "/git"})
	}
	if all || only["configs"] {
		machine.Configs = scan.Configs(home, false)
	}
	return machine
}

// scoped collects the set category flags; empty means all.
func scoped(categories map[string]*bool) map[string]bool {
	only := map[string]bool{}
	for name, set := range categories {
		if *set {
			only[name] = true
		}
	}
	return only
}

// filterCategories drops changes outside the asked-for categories.
func filterCategories(changes []diff.Change, only map[string]bool) []diff.Change {
	if len(only) == 0 {
		return changes
	}
	alias := map[string]string{types.BlueprintTypeFiles: "configs"}
	var kept []diff.Change
	for _, change := range changes {
		name := change.Category
		if a, ok := alias[name]; ok {
			name = a
		}
		if only[name] {
			kept = append(kept, change)
		}
	}
	return kept
}
