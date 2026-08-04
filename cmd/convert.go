package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/convert"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/spf13/cobra"
)

// newConvertCmd is the cobra wiring for `rwr convert`; the implementation
// lives in internal/convert.
func newConvertCmd() *cobra.Command {
	var (
		toFormat string
		migrate  bool
		write    bool
	)

	convertCmd := &cobra.Command{
		Use:   "convert [path]",
		Short: "Convert a blueprint tree between formats, or migrate deprecated constructs",
		Long: `Convert every blueprint, init, bootstrap, and manifest file in a tree to
another format (--to yaml|json|toml|cue), or rewrite deprecated constructs to
their current equivalents (--migrate).

Dry-run by default: nothing is written without --write. Comments are NOT
preserved across formats - the command warns per file that carries them.
Template placeholders survive as quoted strings; a file whose templates make
it unparseable is reported and skipped, never mangled.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			if toFormat == "" && !migrate {
				return fmt.Errorf("nothing to do: pass --to <format> and/or --migrate")
			}
			if toFormat != "" {
				canonical, err := helpers.CanonicalFormat(toFormat)
				if err != nil {
					return err
				}
				toFormat = canonical
			}
			return convert.Run(cmd.OutOrStdout(), root, toFormat, migrate, write)
		},
	}

	convertCmd.Flags().StringVar(&toFormat, "to", "", "Target format: yaml, json, toml, or cue")
	convertCmd.Flags().BoolVar(&migrate, "migrate", false, "Rewrite deprecated constructs (init-file inline sections) to their current form")
	convertCmd.Flags().BoolVar(&write, "write", false, "Apply the changes (default is a dry run)")
	return convertCmd
}
