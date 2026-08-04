package cmd

import (
	"fmt"
	"os"
	"sort"

	"charm.land/huh/v2"
	"github.com/fynxlabs/rwr/internal/capture"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newCaptureCmd is wiring; selection defaults and generation live in
// internal/capture.
func newCaptureCmd(app *AppConfig) *cobra.Command {
	var (
		format   string
		all      bool
		manifest bool
	)

	captureCmd := &cobra.Command{
		Use:   "capture [dir]",
		Short: "Turn this handcrafted machine into a blueprint tree",
		Long: `Scan the machine — explicitly-installed packages across every detected
package manager, dotfiles and ~/.config entries, enabled services, git
checkouts — pick what to keep on a per-category form, and generate a
validated blueprint tree from the selection. --all skips the form and takes
the defaults; --manifest adds a root manifest matched to this machine.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "rwr-blueprints"
			if len(args) == 1 {
				dir = args[0]
			}
			canonical, err := helpers.CanonicalFormat(defaultString(format, "cue"))
			if err != nil {
				return err
			}

			home, _ := os.UserHomeDir() //nolint:errcheck // empty home degrades templating only
			findings := capture.Findings{
				Packages: scan.Packages(system.GetAvailableProviders()),
				Configs:  scan.Configs(home, false),
				Services: scan.Services(),
				Git:      scan.GitCheckouts([]string{home + "/git"}),
				Home:     home,
			}

			selection := capture.Defaults(findings)
			if !all {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("capture is interactive; use --all to take the defaults without a terminal")
				}
				if err := selectionForm(findings, &selection); err != nil {
					return err
				}
			}

			return capture.Generate(cmd.OutOrStdout(), dir, canonical, selection, findings, manifest, app.OSInfo)
		},
	}

	captureCmd.Flags().StringVar(&format, "format", "cue", "Tree format: cue, yaml, json, or toml")
	captureCmd.Flags().BoolVar(&all, "all", false, "Take the default selection without the form")
	captureCmd.Flags().BoolVar(&manifest, "manifest", false, "Write a root manifest matched to this machine")
	return captureCmd
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// selectionForm is the per-category multi-select walk over the findings.
func selectionForm(findings capture.Findings, selection *capture.Selection) error {
	var groups []*huh.Group

	providers := make([]string, 0, len(findings.Packages))
	byProvider := map[string]scan.PackageResult{}
	for _, result := range findings.Packages {
		providers = append(providers, result.Provider)
		byProvider[result.Provider] = result
	}
	sort.Strings(providers)

	packagePicks := map[string]*[]string{}
	for _, provider := range providers {
		result := byProvider[provider]
		options := make([]huh.Option[string], 0, len(result.Names))
		preselect := !result.Unfiltered
		for _, name := range result.Names {
			options = append(options, huh.NewOption(name, name).Selected(preselect))
		}
		title := fmt.Sprintf("Packages — %s (%d found)", provider, len(result.Names))
		if result.Unfiltered {
			title += " [full list: no explicit query, nothing pre-selected]"
		}
		picked := &[]string{}
		packagePicks[provider] = picked
		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[string]().Title(title).Options(options...).Value(picked).Filterable(true),
		))
	}

	configOptions := make([]huh.Option[scanConfigKey], 0, len(findings.Configs))
	for i, config := range findings.Configs {
		label := config.Rel
		if scan.SecretShaped(config.Rel) {
			label += "  [secret-shaped: review before committing]"
		}
		configOptions = append(configOptions, huh.NewOption(label, scanConfigKey(i)).
			Selected(config.Known && !scan.SecretShaped(config.Rel)))
	}
	var configPicks []scanConfigKey
	if len(configOptions) > 0 {
		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[scanConfigKey]().
				Title(fmt.Sprintf("Configs (%d found)", len(configOptions))).
				Options(configOptions...).Value(&configPicks).Filterable(true),
		))
	}

	var servicePicks []string
	if len(findings.Services) > 0 {
		options := make([]huh.Option[string], 0, len(findings.Services))
		for _, service := range findings.Services {
			options = append(options, huh.NewOption(service, service)) // none pre-selected: mostly distro plumbing
		}
		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(fmt.Sprintf("Services (%d enabled — most are distro plumbing)", len(findings.Services))).
				Options(options...).Value(&servicePicks).Filterable(true),
		))
	}

	var gitPicks []int
	if len(findings.Git) > 0 {
		options := make([]huh.Option[int], 0, len(findings.Git))
		for i, checkout := range findings.Git {
			options = append(options, huh.NewOption(checkout.Path, i).Selected(true))
		}
		groups = append(groups, huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title(fmt.Sprintf("Git checkouts (%d found)", len(findings.Git))).
				Options(options...).Value(&gitPicks).Filterable(true),
		))
	}

	if len(groups) == 0 {
		return fmt.Errorf("the scan found nothing to capture")
	}
	if err := huh.NewForm(groups...).Run(); err != nil {
		return err
	}

	selection.Packages = map[string][]string{}
	for provider, picked := range packagePicks {
		if len(*picked) > 0 {
			selection.Packages[provider] = *picked
		}
	}
	selection.Configs = nil
	for _, key := range configPicks {
		config := findings.Configs[int(key)]
		if scan.SecretShaped(config.Rel) {
			fmt.Fprintf(os.Stderr, "warning: %s is secret-shaped — captured because you selected it; review before committing\n", config.Rel)
		}
		selection.Configs = append(selection.Configs, config)
	}
	selection.Services = servicePicks
	selection.Git = nil
	for _, index := range gitPicks {
		selection.Git = append(selection.Git, findings.Git[index])
	}
	return nil
}

// scanConfigKey indexes findings.Configs through the form.
type scanConfigKey int
