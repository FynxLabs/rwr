package cmd

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/processors"
	"github.com/fynxlabs/rwr/internal/tui"
	"github.com/spf13/cobra"
)

func newRunCmd(app *AppConfig) *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run <processor>",
		Short: "Run a single processor",
		Long: `Run a single processor.

"rwr run" on its own lists the processors, like a task runner. Name one to
run it — "rwr run packages" — or use the shorthand straight off the root:
"rwr packages". To run everything: "rwr run all", or "rwr all" from the root.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bare `rwr run` lists the processors, like a bare `mise run`.
			if err := cmd.Help(); err != nil {
				return err
			}
			if len(args) > 0 {
				return fmt.Errorf("unknown processor %q (see the list above)", args[0])
			}
			return nil
		},
	}

	// `rwr run all` runs everything — the same task `rwr all` names at the root.
	runCmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Run all processors",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEverything(app)
		},
	})

	for _, p := range runProcessors {
		runCmd.AddCommand(&cobra.Command{
			Use:   p.use,
			Short: p.short,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runOneProcessor(app, p)
			},
		})
	}

	return runCmd
}

// runEverything is the whole-tree run behind `rwr all`. On a real terminal
// it runs under the dashboard; everywhere else the output is byte-identical
// to the pre-TUI stream.
func runEverything(app *AppConfig) error {
	if tui.Active(app.NoTUI) {
		return runWithTUI(app, nil)
	}
	return runEverythingHeadless(app, nil)
}

// githubAuthIfRequested runs the --gh-auth device flow and records the token.
func githubAuthIfRequested(app *AppConfig) error {
	if !app.GHAuth {
		return nil
	}
	token, err := processors.AuthenticateWithGitHub(app.InitConfig)
	if err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}
	app.GHAPIToken = token
	app.InitConfig.Variables.Flags.GHAPIToken = token
	return nil
}

// runOneProcessor dispatches a single processor, shared by the `rwr run <p>`
// subcommands and the root-level shorthand (`rwr packages`, mise-style).
func runOneProcessor(app *AppConfig, p runProcessorSpec) error {
	if p.githubAuth {
		if err := githubAuthIfRequested(app); err != nil {
			return err
		}
	}
	if p.bootstrap {
		return processors.RunBootstrap(app.InitConfig, app.OSInfo)
	}
	// Same TUI, one strip block: the panel takes the freed vertical space.
	if tui.Active(app.NoTUI) {
		return runWithTUI(app, []string{p.blueprint})
	}
	return processors.All(app.InitConfig, app.OSInfo, []string{p.blueprint})
}

// selectedProcessorsFor maps the invoked command to the blueprint types this
// run will execute, so credential resolution can skip credentials scoped to
// processors that are not running. Nil means all.
func selectedProcessorsFor(cmd *cobra.Command, args []string) []string {
	name := cmd.Name()
	// Root-level shorthand: `rwr packages` resolves at the root with args.
	if cmd.Parent() == nil && len(args) > 0 {
		name = args[0]
	}
	if p, ok := processorShorthand(name); ok {
		if p.bootstrap {
			return []string{"bootstrap"}
		}
		return []string{p.blueprint}
	}
	return nil
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

// runProcessorSpec maps each subcommand to the blueprint type it dispatches.
type runProcessorSpec struct {
	use       string
	short     string
	blueprint string
	// ssh_keys is the one processor with pre-work: --gh-auth runs the GitHub
	// device flow before the processor needs the token.
	githubAuth bool
	// bootstrap dispatches to the standalone bootstrap entry: the generic
	// processors.All path never runs bootstrap, and an explicit invocation
	// bypasses the run-once marker.
	bootstrap bool
}

var runProcessors = []runProcessorSpec{
	{use: "bootstrap", short: "Run the bootstrap processor (ignores the run-once marker)", bootstrap: true},
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
