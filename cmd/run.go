package cmd

import (
	"fmt"

	"charm.land/log/v2"

	"github.com/fynxlabs/rwr/internal/processors"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [processor]",
	Short: "Run the blueprints — everything, or a single processor",
	Long: `Run the blueprint tree.

"rwr run" on its own runs everything, the same as "rwr all". Name a processor
to run just that piece: "rwr run packages", "rwr run files".`,
	// Bare `rwr run` is the landing command: it runs the whole tree. It used
	// to print help and error, while the docs told people to run it.
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			if err := cmd.Help(); err != nil {
				return err
			}
			return fmt.Errorf("unknown processor %q (see the list above)", args[0])
		}
		return runEverything()
	},
}

// runEverything is the whole-tree run both `rwr run` and `rwr all` dispatch.
func runEverything() error {
	// Handle GitHub OAuth authentication if --gh-auth flag is set
	if ghAuth {
		token, err := processors.AuthenticateWithGitHub(initConfig)
		if err != nil {
			return fmt.Errorf("GitHub authentication failed: %w", err)
		}
		// Update the token in both global var and initConfig
		ghApiToken = token
		initConfig.Variables.Flags.GHAPIToken = token
	}

	log.Debugf("ForceBootstrap: %v", initConfig.Variables.Flags.ForceBootstrap)
	if err := processors.All(initConfig, osInfo, nil); err != nil {
		return fmt.Errorf("error running all processors: %w", err)
	}
	return nil
}

// runProcessors maps each subcommand to the blueprint type it dispatches. One
// table instead of ten hand-written near-identical command declarations: the
// subcommand names and behavior are unchanged.
type runProcessorSpec struct {
	use       string
	short     string
	blueprint string
	// ssh_keys is the one processor with pre-work: --gh-auth runs the GitHub
	// device flow before the processor needs the token.
	githubAuth bool
}

var runProcessors = []runProcessorSpec{
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

// runOneProcessor dispatches a single processor, shared by the `rwr run <p>`
// subcommands and the root-level shorthand (`rwr packages`, mise-style).
func runOneProcessor(p runProcessorSpec) error {
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
}

// processorShorthand resolves a root-level argument to a processor, so
// `rwr packages` works like `rwr run packages` — the processor names are not
// part of the prime command namespace, exactly like a task runner's tasks.
func processorShorthand(name string) (runProcessorSpec, bool) {
	for _, p := range runProcessors {
		if p.use == name {
			return p, true
		}
	}
	return runProcessorSpec{}, false
}

func init() {
	rootCmd.AddCommand(runCmd)

	for _, p := range runProcessors {
		runCmd.AddCommand(&cobra.Command{
			Use:   p.use,
			Short: p.short,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runOneProcessor(p)
			},
		})
	}
}
