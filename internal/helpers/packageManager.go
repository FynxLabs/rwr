package helpers

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"
)

func GetPackageManagerInfo(osInfo *types.OSInfo, pm string) (types.PackageManagerInfo, error) {
	if prov, exists := providers.GetProvider(pm); exists {
		pmInfo := providers.GetPackageManagerInfo(prov, prov.BinPath)
		return types.PackageManagerInfo{
			Name:     pmInfo.Name,
			Bin:      pmInfo.Bin,
			List:     pmInfo.List,
			Search:   pmInfo.Search,
			Install:  pmInfo.Install,
			Remove:   pmInfo.Remove,
			Update:   pmInfo.Update,
			Clean:    pmInfo.Clean,
			Elevated: pmInfo.Elevated,
		}, nil
	}
	return types.PackageManagerInfo{}, fmt.Errorf("unsupported package manager: %s", pm)
}

func InstallOpenSSL(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	// Get provider for default package manager
	provider, exists := providers.GetProvider(osInfo.PackageManager.Default.Name)
	if !exists {
		return fmt.Errorf("no package manager available for OpenSSL installation")
	}

	// Get OpenSSL packages for this provider
	packages := provider.CorePackages["openssl"]
	if len(packages) == 0 {
		return fmt.Errorf("no OpenSSL packages defined for %s", provider.Name)
	}

	// Install each package
	log.Debugf("Installing OpenSSL packages with %s: %v", provider.Name, packages)
	installCmd := types.Command{
		Exec:     fmt.Sprintf("%s %s", provider.BinPath, provider.Commands.Install),
		Args:     packages,
		Elevated: provider.Elevated,
	}

	if err := RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error installing OpenSSL packages: %v", err)
	}

	return nil
}

// InstallBuildEssentials installs build essentials on the system.
func InstallBuildEssentials(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	// Get provider for default package manager
	provider, exists := providers.GetProvider(osInfo.PackageManager.Default.Name)
	if !exists {
		return fmt.Errorf("no package manager available for build essentials installation")
	}

	// Get build essential packages for this provider
	packages := provider.CorePackages["build-essentials"]
	if len(packages) == 0 {
		return fmt.Errorf("no build essential packages defined for %s", provider.Name)
	}

	// Install each package
	log.Debugf("Installing build essential packages with %s: %v", provider.Name, packages)
	installCmd := types.Command{
		Exec:     fmt.Sprintf("%s %s", provider.BinPath, provider.Commands.Install),
		Args:     packages,
		Elevated: provider.Elevated,
	}

	if err := RunCommand(installCmd, initConfig.Variables.Flags.Debug); err != nil {
		return fmt.Errorf("error installing build essential packages: %v", err)
	}

	return nil
}
