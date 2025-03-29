package processors

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
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

	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	// Get available providers
	available := providers.GetAvailableProviders()
	if len(available) == 0 {
		return fmt.Errorf("no package managers available")
	}

	// Process each package
	for _, pkg := range packages.Packages {
		// Get provider
		var provider *providers.Provider
		var exists bool

		if pkg.PackageManager != "" {
			// Use specified package manager
			provider, exists = providers.GetProvider(pkg.PackageManager)
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
			// Build command
			var cmdStr string
			switch pkg.Action {
			case "install":
				cmdStr = fmt.Sprintf("%s %s %s", provider.BinPath, provider.Commands.Install, name)
			case "remove":
				cmdStr = fmt.Sprintf("%s %s %s", provider.BinPath, provider.Commands.Remove, name)
			default:
				log.Warnf("Unknown action %s for package %s", pkg.Action, name)
				continue
			}

			// Add any additional arguments
			if len(pkg.Args) > 0 {
				cmdStr = fmt.Sprintf("%s %s", cmdStr, strings.Join(pkg.Args, " "))
			}

			// Execute command
			cmd := exec.Command("sh", "-c", cmdStr)
			if out, err := cmd.CombinedOutput(); err != nil {
				log.Warnf("Error %s package %s: %v\n%s", pkg.Action, name, err, string(out))
				continue
			}

			log.Infof("Successfully %sd package %s", pkg.Action, name)
		}
	}

	return nil
}
