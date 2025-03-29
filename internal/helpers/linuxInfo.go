package helpers

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"

	"github.com/charmbracelet/log"
)

func SetLinuxDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Linux package manager details.")

	// Initialize providers
	providersPath, err := providers.GetProvidersPath()
	if err != nil {
		return fmt.Errorf("error getting providers path: %w", err)
	}

	if err := providers.LoadProviders(providersPath); err != nil {
		return fmt.Errorf("error loading providers: %w", err)
	}

	// Initialize package manager map
	if osInfo.PackageManager.Managers == nil {
		osInfo.PackageManager.Managers = make(map[string]types.PackageManagerInfo)
	}

	// Get available providers
	available := providers.GetAvailableProviders()

	// Add all available Linux package managers
	for name, prov := range available {
		// Skip if not a Linux provider
		if !Contains(prov.Detection.Distributions, "linux") {
			continue
		}

		if binPath, err := GetBinPath(prov.Detection.Binary); err == nil {
			pmInfo := providers.GetPackageManagerInfo(prov, binPath)
			osInfo.PackageManager.Managers[name] = types.PackageManagerInfo{
				Name:     pmInfo.Name,
				Bin:      pmInfo.Bin,
				List:     pmInfo.List,
				Search:   pmInfo.Search,
				Install:  pmInfo.Install,
				Remove:   pmInfo.Remove,
				Update:   pmInfo.Update,
				Clean:    pmInfo.Clean,
				Elevated: pmInfo.Elevated,
			}
			log.Debugf("Added package manager: %s", name)
		}
	}

	// Get default from OS release
	defaultPackageManager := providers.GetDefaultProviderFromOSRelease()
	if defaultPackageManager != "" && osInfo.PackageManager.Managers[defaultPackageManager].Bin != "" {
		osInfo.PackageManager.Default = osInfo.PackageManager.Managers[defaultPackageManager]
		log.Infof("Set %s as default package manager from OS release", defaultPackageManager)
	} else if osInfo.System.OSFamily == "arch" {
		// For Arch Linux, prioritize AUR helpers (in order of preference)
		aurHelpers := []string{"paru", "yay", "trizen", "aura", "pamac"}
		for _, helper := range aurHelpers {
			if pm, exists := osInfo.PackageManager.Managers[helper]; exists && pm.Bin != "" {
				osInfo.PackageManager.Default = pm
				log.Infof("Set AUR helper %s as default package manager", helper)
				break
			}
		}
	} else {
		// Otherwise use first available package manager as default
		for _, pm := range osInfo.PackageManager.Managers {
			osInfo.PackageManager.Default = pm
			log.Infof("Set %s as default package manager", pm.Name)
			break
		}
	}

	// Debug log the final default package manager
	if osInfo.PackageManager.Default.Name != "" {
		log.Infof("Final default package manager: %s", osInfo.PackageManager.Default.Name)
	} else {
		log.Warn("No default package manager set")
	}

	return nil
}
