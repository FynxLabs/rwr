package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/tui"
	"github.com/fynxlabs/rwr/internal/types"
)

// isManifestPath reports whether a resolved init source is a repo manifest
// rather than an init file.
func isManifestPath(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return base == "manifest"
}

// selectFromManifest turns a manifest path into the selected entry's init
// file path. Selection per the design: --config-name always wins; zero
// matches errors listing every entry and its matchers; exactly one match is
// used and logged; multiple matches prompt on a TTY and error headless.
func selectFromManifest(app *AppConfig, manifestPath string) (string, error) {
	manifest, err := helpers.LoadManifest(manifestPath)
	if err != nil {
		return "", err
	}
	root := filepath.Dir(manifestPath)

	entryInit := func(entry types.ManifestEntry) string {
		return filepath.Join(root, entry.Init)
	}

	if app.ConfigName != "" {
		for _, entry := range manifest.Configurations {
			if entry.Name == app.ConfigName {
				return entryInit(entry), nil
			}
		}
		return "", fmt.Errorf("manifest has no configuration named %q; it has: %s", app.ConfigName, entryNames(manifest.Configurations))
	}

	// The matchers filter against the detected system; DetectOS needs only
	// SetPaths, which is why selection can run before init.
	if err := system.SetPaths(); err != nil {
		return "", fmt.Errorf("error setting paths: %w", err)
	}
	osInfo := system.DetectOS()
	matched := helpers.MatchManifest(manifest, osInfo)

	switch len(matched) {
	case 0:
		var lines []string
		for _, entry := range manifest.Configurations {
			lines = append(lines, fmt.Sprintf("  %s (os=%s distro=%s family=%s arch=%s)", entry.Name, entry.OS, entry.Distro, entry.Family, entry.Arch))
		}
		return "", fmt.Errorf("no manifest configuration matches this machine (%s/%s/%s); entries:\n%s\nselect one explicitly with --config-name",
			osInfo.System.OS, osInfo.System.OSFamily, osInfo.System.OSArch, strings.Join(lines, "\n"))
	case 1:
		log.Infof("Using manifest configuration %q (matched this machine)", matched[0].Name)
		return entryInit(matched[0]), nil
	}

	// Prefer a declared default among the matches before prompting.
	for _, entry := range matched {
		if entry.Default {
			log.Infof("Using manifest configuration %q (default among %d matches)", entry.Name, len(matched))
			return entryInit(entry), nil
		}
	}

	if !tui.Active(app.NoTUI) {
		return "", fmt.Errorf("%d manifest configurations match this machine (%s); scripts and CI must pick one with --config-name",
			len(matched), entryNames(matched))
	}

	// The selection frame, rendered before resolve stage 1 ever runs:
	// matched entries first, the rest after — selectable, because matchers
	// are hints and the operator may know better.
	isMatched := map[string]bool{}
	for _, entry := range matched {
		isMatched[entry.Name] = true
	}
	options := make([]huh.Option[string], 0, len(manifest.Configurations))
	for _, entry := range matched {
		options = append(options, huh.NewOption(entry.Name+"  ("+entry.Init+")", entry.Name))
	}
	for _, entry := range manifest.Configurations {
		if !isMatched[entry.Name] {
			options = append(options, huh.NewOption(entry.Name+"  ("+entry.Init+") — not matched", entry.Name))
		}
	}
	var chosen string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Which configuration?").Options(options...).Value(&chosen),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("configuration selection cancelled: %w", err)
	}
	for _, entry := range manifest.Configurations {
		if entry.Name == chosen {
			return entryInit(entry), nil
		}
	}
	return "", fmt.Errorf("selection %q not found", chosen)
}

func entryNames(entries []types.ManifestEntry) string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name
	}
	return strings.Join(names, ", ")
}
