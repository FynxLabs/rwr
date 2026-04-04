package system

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// GetPackageManager retrieves the configuration for a named package manager
// from the available providers. Returns an error if the provider is not found.
func GetPackageManager(pm string) (types.PackageManagerInfo, error) {
	if prov, exists := GetProviderWithAlternatives(pm); exists {
		return GetPackageManagerInfo(prov, prov.BinPath), nil
	}
	return types.PackageManagerInfo{}, fmt.Errorf("unsupported package manager: %s", pm)
}

// InstallOpenSSL ensures OpenSSL is installed on the system, using the
// default package manager to install it if not already present.
func InstallOpenSSL(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Check if OpenSSL is already installed
	opensslTool := FindTool("openssl")
	if opensslTool.Exists {
		log.Infof("OpenSSL is already installed at %s", opensslTool.Bin)
		return nil
	}

	// Initialize providers
	if err := InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Check if default package manager is set
	if osInfo.PackageManager.Default.Name == "" {
		log.Warnf("No default package manager set, skipping OpenSSL installation")
		return nil
	}

	// Get provider for default package manager with alternatives applied
	provider, exists := GetProviderWithAlternatives(osInfo.PackageManager.Default.Name)
	if !exists {
		log.Warnf("No provider found for %s, skipping OpenSSL installation", osInfo.PackageManager.Default.Name)
		return nil
	}

	// Get OpenSSL packages for this provider
	packages := provider.CorePackages["openssl"]
	if len(packages) == 0 {
		log.Warnf("No OpenSSL packages defined for %s, skipping installation", provider.Name)
		return nil
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

// InstallBuildEssentials installs common build tools (make, gcc, cmake) using
// the default package manager, skipping any that are already installed.
func InstallBuildEssentials(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Check for common build tools to see if they're already installed
	makeExists := FindTool("make").Exists
	gccExists := FindTool("gcc").Exists
	cmakeExists := FindTool("cmake").Exists

	if makeExists && gccExists && cmakeExists {
		log.Infof("Build essentials already installed (make, gcc, cmake found)")
		return nil
	}

	// Initialize providers
	if err := InitProviders(); err != nil {
		return fmt.Errorf("error initializing providers: %w", err)
	}

	// Check if default package manager is set
	if osInfo.PackageManager.Default.Name == "" {
		log.Warnf("No default package manager set, skipping build essentials installation")
		return nil
	}

	// Get provider for default package manager with alternatives applied
	provider, exists := GetProviderWithAlternatives(osInfo.PackageManager.Default.Name)
	if !exists {
		log.Warnf("No provider found for %s, skipping build essentials installation", osInfo.PackageManager.Default.Name)
		return nil
	}

	// Get build essential packages for this provider
	packages := provider.CorePackages["build-essentials"]
	if len(packages) == 0 {
		log.Warnf("No build essential packages defined for %s, skipping installation", provider.Name)
		return nil
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
