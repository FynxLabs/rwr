package system

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetMacOSDetails sets macOS-specific system details
func SetMacOSDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting macOS package manager details.")

	// Initialize package manager map
	if osInfo.PackageManager.Managers == nil {
		osInfo.PackageManager.Managers = make(map[string]types.PackageManagerInfo)
	}

	// Get available providers
	available := GetAvailableProviders()

	// Add all available macOS package managers
	for name, prov := range available {
		// Skip if not a macOS provider
		if !Contains(prov.Detection.Distributions, "darwin") {
			continue
		}

		if tool := FindTool(prov.Detection.Binary); tool.Exists {
			pmInfo := GetPackageManagerInfo(prov, tool.Bin)
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

	// Set default package manager (prefer Homebrew)
	if pm, exists := osInfo.PackageManager.Managers["brew"]; exists {
		osInfo.PackageManager.Default = pm
		log.Infof("Set Homebrew as default package manager")
	} else if pm, exists := osInfo.PackageManager.Managers["macports"]; exists {
		osInfo.PackageManager.Default = pm
		log.Infof("Set MacPorts as default package manager")
	} else {
		// Use first available package manager as default
		for _, pm := range osInfo.PackageManager.Managers {
			osInfo.PackageManager.Default = pm
			log.Infof("Set %s as default package manager", pm.Name)
			break
		}
	}

	return nil
}

// getDarwinVersion returns the macOS version
func getDarwinVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	out, err := cmd.Output()
	if err != nil {
		log.Warnf("Error getting macOS version: %v", err)
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}
