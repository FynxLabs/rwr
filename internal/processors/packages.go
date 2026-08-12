package processors

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessPackages installs or removes packages based on blueprint definitions.
// It supports two modes:
//   - If data is provided: unmarshals it into package definitions
//   - If packages is provided: uses the provided package list directly
//
// The function resolves import directives recursively (with circular detection),
// filters packages based on active profiles from initConfig, detects available
// package managers for the current OS, and executes install/remove commands.
//
// Returns an error if no package managers are available or if unmarshaling fails.
// Individual package installation errors are logged but do not stop processing.
func ProcessPackages(data []byte, packages *types.PackagesData, blueprintDir string, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// If data is provided, unmarshal it
	if data != nil {
		var pkgData types.PackagesData
		if err := helpers.DecodeBlueprintInto(data, format, types.BlueprintTypePackages,
			helpers.TreeSchemaVersion(initConfig), &pkgData); err != nil {
			return fmt.Errorf("error unmarshaling package blueprint: %w", err)
		}
		packages = &pkgData
	}

	// If no packages provided, nothing to do
	if packages == nil || len(packages.Packages) == 0 {
		return nil
	}

	// Process imports and merge imported packages
	allPackages, err := helpers.ResolveImports(packages.Packages, blueprintDir,
		func(item types.Package) string { return item.Import },
		func(data []byte, fileFormat string) ([]types.Package, error) {
			var d types.PackagesData
			if err := helpers.DecodeBlueprintInto(data, fileFormat, types.BlueprintTypePackages,
				helpers.TreeSchemaVersion(initConfig), &d); err != nil {
				return nil, err
			}
			return d.Packages, nil
		}, format)
	if err != nil {
		return fmt.Errorf("error processing package imports: %w", err)
	}

	// Update packages with merged list
	packages.Packages = allPackages

	// Initialize providers if needed
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Get available providers
	available := system.GetAvailableProviders()
	if len(available) == 0 {
		return fmt.Errorf("no package managers available - check debug logs for detailed provider detection information. Common issues: missing binaries in PATH, missing config files, or unsupported platform")
	}

	// Filter packages based on active profiles
	filteredPackages := helpers.FilterByProfiles(packages.Packages, initConfig.Variables.Flags.Profiles)

	log.Debugf("Filtering packages: %d total, %d matching active profiles %v",
		len(packages.Packages), len(filteredPackages), initConfig.Variables.Flags.Profiles)

	// Resolve each package's provider and names up front so the lanes know
	// their denominators before the first install runs.
	type packageUnit struct {
		pkg      types.Package
		provider *types.Provider
		names    []string
	}
	var units []packageUnit
	track := newProgress(types.BlueprintTypePackages)
	for _, pkg := range filteredPackages {
		// Get provider
		var provider *types.Provider
		var exists bool

		// A package entry that cannot reach a package manager is a FAILURE,
		// not a skip. Blueprint trees are per-OS, so a declared
		// package_manager is a demand: skipping here meant a machine without
		// cargo "successfully" ran a tree whose init file exists to install
		// cargo's packages, and the run exited 0 having installed none of
		// them. The ledger keeps the run going but puts it in the exit code.
		subject := pkg.Name
		if subject == "" && len(pkg.Names) > 0 {
			subject = strings.Join(pkg.Names, " ")
		}
		if pkg.PackageManager != "" {
			// Use specified package manager
			provider, exists = system.GetProvider(pkg.PackageManager)
			if !exists {
				recordFailure("packages", subject,
					fmt.Errorf("required package manager %q is not available on this system; declare it under packageManagers in the init file (action: install) or install it manually", pkg.PackageManager))
				track.item(pkg.PackageManager, subject, pkg.Action, types.StatusFailed, "package manager not available", 0)
				continue
			}
		} else {
			provider, exists = defaultProviderFor(osInfo, available)
			if !exists {
				recordFailure("packages", subject,
					errors.New("no package manager available for this entry; declare one under packageManagers in the init file (action: install) or set package_manager on the entry"))
				track.item("", subject, pkg.Action, types.StatusFailed, "no package manager available", 0)
				continue
			}
		}

		// Get package names. `names` wins over `name` when both are set,
		// matching files and fonts - packages had the precedence backwards,
		// so the same both-declared entry meant different things per type.
		var names []string
		if len(pkg.Names) > 0 {
			names = pkg.Names
			if pkg.Name != "" {
				log.Warnf("Package entry declares both name (%q) and names; processing the names list and ignoring name", pkg.Name)
			}
		} else if pkg.Name != "" {
			names = []string{pkg.Name}
		}

		units = append(units, packageUnit{pkg: pkg, provider: provider, names: names})
		track.expect(provider.Name, len(names))
	}

	for _, unit := range units {
		pkg, provider := unit.pkg, unit.provider

		// The provider stamp fills the log view's provider column for every
		// line this unit's work produces (rwr's own and captured output).
		reporting.SetCurrentProvider(provider.Name)

		// Process each package
		for _, name := range unit.names {
			// Build command arguments
			var args []string
			switch pkg.Action {
			case "install":
				args = append(args, strings.Fields(provider.Commands.Install)...)
			case "remove":
				args = append(args, strings.Fields(provider.Commands.Remove)...)
			default:
				recordFailure("packages", name, fmt.Errorf("unknown action %q", pkg.Action))
				track.item(provider.Name, name, pkg.Action, types.StatusFailed, "unknown action", 0)
				continue
			}

			// A name beginning with "-" is read as an option by every package
			// manager, not as a package: "--allow-downgrades", "-U <url>". Commands
			// are argv-exec'd so this is not shell injection, but it still lets a
			// blueprint change what the elevated package manager does.
			if strings.HasPrefix(name, "-") {
				recordFailure("packages", name, errors.New("package name may not begin with '-'; it would be read as an option by the package manager"))
				track.item(provider.Name, name, pkg.Action, types.StatusFailed, "name begins with '-'", 0)
				continue
			}

			args = append(args, name)

			// Add any additional arguments
			if len(pkg.Args) > 0 {
				args = append(args, pkg.Args...)
			}

			// Execute command directly with environment variables
			cmd := types.Command{
				Exec: provider.BinPath,
				Args: args,
				// The provider decides whether its package manager needs elevation;
				// a blueprint may ask for it on top (a user-scoped manager invoked
				// against a system path), but may not take it away.
				Elevated:  provider.Elevated || pkg.Elevated,
				Variables: provider.Environment,
				// Terminal handover only on an explicit per-item
				// `interactive: true`. Routing every package through the
				// terminal suspended the TUI per package and splattered raw
				// package-manager output across the dashboard - and the one
				// legitimate need, sudo's password prompt, is served by
				// ensureSudoCredentials validating before captured elevated
				// commands run.
				Interactive: helpers.ResolveInteractive(pkg.Interactive, false),
			}
			started := time.Now()
			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				recordFailure("packages", name, fmt.Errorf("%s failed: %w", pkg.Action, err))
				track.item(provider.Name, name, pkg.Action, types.StatusFailed, err.Error(), time.Since(started))
				continue
			}

			track.item(provider.Name, name, pkg.Action, types.StatusOK, "", time.Since(started))
			log.Infof("Successfully %s package %s via %s", pastTense(pkg.Action), name, provider.Name)
		}
	}
	reporting.SetCurrentProvider("")

	return nil
}

// defaultProviderFor picks the provider to use for a package that did not name a
// package_manager.
//
// It prefers the default resolved during OS detection (which honours
// /etc/os-release and, on Arch, the AUR helper preference order), and otherwise
// falls back to the alphabetically first available provider. The fallback is
// sorted because Go randomizes map iteration: selecting "whatever the map yields
// first" meant an unqualified package could be installed by a different package
// manager on every run.
// pastTense renders a package action for the success message. The old
// "%sed" format produced "removeed": actions ending in "e" only take a "d".
func pastTense(action string) string {
	if strings.HasSuffix(action, "e") {
		return action + "d"
	}
	return action + "ed"
}

func defaultProviderFor(osInfo *types.OSInfo, available map[string]*types.Provider) (*types.Provider, bool) {
	if osInfo != nil {
		if name := osInfo.PackageManager.Default.Name; name != "" {
			if provider, ok := available[name]; ok {
				return provider, true
			}
		}
	}

	names := make([]string, 0, len(available))
	for name := range available {
		names = append(names, name)
	}
	sort.Strings(names)

	if len(names) == 0 {
		return nil, false
	}
	return available[names[0]], true
}
