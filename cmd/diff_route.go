package cmd

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/fynxlabs/rwr/internal/diff"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/cobra"
)

// routeInteractively asks, per change group, where the additions land — a
// destination the tree itself offers, or skip — and writes the accepted
// edits. Machine-specific versus Common is the operator's call; rwr's job
// is to make it once per group.
func routeInteractively(cmd *cobra.Command, changes []diff.Change, tree string) error {
	out := cmd.OutOrStdout()

	byGroup := map[string][]diff.Change{}
	for _, change := range changes {
		if change.Removal {
			continue // removals are reported, never auto-deleted from blueprints
		}
		switch change.Category {
		case types.BlueprintTypePackages, types.BlueprintTypeServices:
			byGroup[change.Category+"\x00"+change.Provider] = append(byGroup[change.Category+"\x00"+change.Provider], change)
		}
	}
	if len(byGroup) == 0 {
		helpers.Say(out, "Nothing routable: only packages and services route today; use --format for the rest.\n")
		return nil
	}

	routed := 0
	for key, group := range byGroup {
		category := group[0].Category
		destinations, err := diff.Destinations(tree, category)
		if err != nil {
			helpers.Say(out, "skip %s: %v\n", key, err)
			continue
		}

		names := make([]string, 0, len(group))
		for _, change := range group {
			names = append(names, change.Name)
		}
		title := fmt.Sprintf("%s: %d addition(s)", category, len(names))
		if group[0].Provider != "" {
			title = fmt.Sprintf("%s (%s): %d addition(s)", category, group[0].Provider, len(names))
		}

		options := make([]huh.Option[string], 0, len(destinations)+1)
		for _, destination := range destinations {
			options = append(options, huh.NewOption(destination, destination))
		}
		options = append(options, huh.NewOption("skip", ""))

		var destination string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Description(fmt.Sprintf("%v", names)).
				Options(options...).
				Value(&destination),
		))
		if err := form.Run(); err != nil {
			return err
		}
		if destination == "" {
			continue
		}

		var entries []map[string]interface{}
		switch category {
		case types.BlueprintTypePackages:
			entries = diff.PackageEntries(group[0].Provider, names)
		case types.BlueprintTypeServices:
			entries = diff.ServiceEntries(names)
		}
		if err := diff.AppendEntries(destination, category, entries); err != nil {
			return err
		}
		helpers.Say(out, "wrote %d entr(ies) to %s\n", len(entries), destination)
		routed++
	}
	if routed > 0 {
		helpers.Say(out, "Run `rwr validate` over the tree, review the diff, and commit.\n")
	}
	return nil
}
