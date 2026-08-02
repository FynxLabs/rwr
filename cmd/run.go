package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/processors"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <processor>",
	Short: "Run an individual processor",
	Long: `Run a single processor instead of the whole blueprint.

"run" is not a command on its own: it needs the name of the processor to run,
for example "rwr run packages" or "rwr run files". To run everything, use
"rwr all".`,
	// A parent command with no action silently prints help and exits 0, which
	// reads like a successful run that did nothing. Make the miss explicit.
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.Help(); err != nil {
			return err
		}
		if len(args) == 0 {
			return fmt.Errorf("run needs a processor to run (see the list above), or use \"rwr all\" to run everything")
		}
		return fmt.Errorf("unknown processor %q (see the list above)", args[0])
	},
}

// runProcessors maps each subcommand to the blueprint type it dispatches. One
// table instead of ten hand-written near-identical command declarations: the
// subcommand names and behavior are unchanged.
var runProcessors = []struct {
	use       string
	short     string
	blueprint string
	// ssh_keys is the one processor with pre-work: --gh-auth runs the GitHub
	// device flow before the processor needs the token.
	githubAuth bool
}{
	{use: "packages", short: "Run packages processor", blueprint: "packages"},
	{use: "repository", short: "Run repository processor", blueprint: "repositories"},
	{use: "services", short: "Run services processor", blueprint: "services"},
	{use: "files", short: "Run files processor", blueprint: "files"},
	{use: "configuration", short: "Run configuration processor", blueprint: "configuration"},
	{use: "users", short: "Run users processor", blueprint: "users"},
	{use: "git", short: "Run git processor", blueprint: "git"},
	{use: "scripts", short: "Run scripts processor", blueprint: "scripts"},
	{use: "ssh_keys", short: "Run SSH key processor", blueprint: "ssh_keys", githubAuth: true},
	{use: "fonts", short: "Run fonts processor", blueprint: "fonts"},
}

func init() {
	rootCmd.AddCommand(runCmd)

	for _, p := range runProcessors {
		runCmd.AddCommand(&cobra.Command{
			Use:   p.use,
			Short: p.short,
			RunE: func(cmd *cobra.Command, args []string) error {
				if p.githubAuth && ghAuth {
					token, err := processors.AuthenticateWithGitHub(initConfig)
					if err != nil {
						return fmt.Errorf("GitHub authentication failed: %w", err)
					}
					// Update the token in both global var and initConfig
					ghApiToken = token
					initConfig.Variables.Flags.GHAPIToken = token
				}
				return processors.All(initConfig, osInfo, []string{p.blueprint})
			},
		})
	}
}
