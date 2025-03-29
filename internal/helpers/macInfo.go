package helpers

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/pkg/providers"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// SetMacOSDetails Sets the package manager details for macOS.
func SetMacOSDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting macOS package manager details.")

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

	// Add all available macOS package managers
	for name, prov := range available {
		// Skip if not a macOS provider
		if !Contains(prov.Detection.Distributions, "darwin") {
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

	// Set default package manager from config if specified
	viperDefault := viper.GetString("packageManager.macos.default")
	if viperDefault != "" && osInfo.PackageManager.Managers[viperDefault].Bin != "" {
		osInfo.PackageManager.Default = osInfo.PackageManager.Managers[viperDefault]
		log.Infof("Set %s as default package manager from config", viperDefault)
	} else {
		// Otherwise use first available package manager as default
		for _, pm := range osInfo.PackageManager.Managers {
			osInfo.PackageManager.Default = pm
			log.Infof("Set %s as default package manager", pm.Name)
			break
		}
	}

	return nil
}

func getDarwinVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("Error getting macOS version: %v", err)
		return "Unknown"
	}
	return strings.TrimSpace(string(output))
}
