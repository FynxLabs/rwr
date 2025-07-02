package processors

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ProcessPackages processes package management operations
func ProcessPackages(data []byte, packages *types.PackagesData, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// If data is provided, unmarshal it
	if data != nil {
		var pkgData types.PackagesData
		if err := helpers.UnmarshalBlueprint(data, format, &pkgData); err != nil {
			return fmt.Errorf("error unmarshaling package blueprint: %w", err)
		}
		packages = &pkgData
	}

	// If no packages provided, nothing to do
	if packages == nil || len(packages.Packages) == 0 {
		return nil
	}

	// Initialize providers if needed
	if err := system.InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Get available providers
	available := system.GetAvailableProviders()
	if len(available) == 0 {
		return fmt.Errorf("no package managers available")
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
			// Use first available provider
			for _, p := range available {
				provider = p
				break
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
				Exec:      provider.BinPath,
				Args:      args,
				Elevated:  provider.Elevated,
				Variables: provider.Environment,
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
