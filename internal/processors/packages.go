package processors

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
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
func ProcessPackages(data []byte, packages *types.PackagesData, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
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
	blueprintDir := initConfig.Init.Location
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

	// Process each filtered package
	for _, pkg := range filteredPackages {
		// Get provider
		var provider *types.Provider
		var exists bool

		if pkg.PackageManager != "" {
			// Use specified package manager
			provider, exists = system.GetProvider(pkg.PackageManager)
			if !exists {
				log.Warnf("Specified package manager %s not available, skipping package %s", pkg.PackageManager, pkg.Name)
				continue
			}
		} else {
			provider, exists = defaultProviderFor(osInfo, available)
			if !exists {
				log.Warnf("No package manager available for package %s, skipping", pkg.Name)
				continue
			}
		}

		// Get package names
		var names []string
		if pkg.Name != "" {
			names = []string{pkg.Name}
		} else {
			names = pkg.Names
		}

		// Process each package
		for _, name := range names {
			// Build command arguments
			var args []string
			switch pkg.Action {
			case "install":
				args = append(args, strings.Fields(provider.Commands.Install)...)
			case "remove":
				args = append(args, strings.Fields(provider.Commands.Remove)...)
			default:
				log.Warnf("Unknown action %s for package %s", pkg.Action, name)
				continue
			}

			// Add package name
			args = append(args, name)

			// Add any additional arguments
			if len(pkg.Args) > 0 {
				args = append(args, pkg.Args...)
			}

			// Execute command directly with environment variables
			cmd := types.Command{
				Exec:        provider.BinPath,
				Args:        args,
				Elevated:    provider.Elevated,
				Variables:   provider.Environment,
				Interactive: helpers.ResolveInteractive(pkg.Interactive, initConfig.Variables.Flags.Interactive),
			}
			if err := system.RunCommand(cmd, initConfig.Variables.Flags.Debug); err != nil {
				log.Warnf("Error %s package %s: %v", pkg.Action, name, err)
				continue
			}

			log.Infof("Successfully %sed package %s via %s", pkg.Action, name, provider.Name)
		}
	}

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
