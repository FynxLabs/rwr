package system

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetWindowsDetails sets Windows-specific system details
func SetWindowsDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Windows package manager details.")

	// Initialize package manager map
	if osInfo.PackageManager.Managers == nil {
		osInfo.PackageManager.Managers = make(map[string]types.PackageManagerInfo)
	}

	// Get available providers
	available := GetAvailableProviders()

	// Add all available Windows package managers
	for name, prov := range available {
		// Skip if not a Windows provider
		if !Contains(prov.Detection.Distributions, "windows") {
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

	// Set default package manager (prefer winget)
	if pm, exists := osInfo.PackageManager.Managers["winget"]; exists {
		osInfo.PackageManager.Default = pm
		log.Infof("Set winget as default package manager")
	} else if pm, exists := osInfo.PackageManager.Managers["chocolatey"]; exists {
		osInfo.PackageManager.Default = pm
		log.Infof("Set Chocolatey as default package manager")
	} else if pm, exists := osInfo.PackageManager.Managers["scoop"]; exists {
		osInfo.PackageManager.Default = pm
		log.Infof("Set Scoop as default package manager")
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

// getWindowsVersion returns the Windows version
func getWindowsVersion() string {
	cmd := exec.Command("cmd", "/c", "ver")
	out, err := cmd.Output()
	if err != nil {
		log.Warnf("Error getting Windows version: %v", err)
		return "Unknown"
	}
	// Output format: Microsoft Windows [Version 10.0.19045.3930]
	version := strings.TrimSpace(string(out))
	if i := strings.Index(version, "[Version "); i != -1 {
		version = version[i+9:]
		if j := strings.Index(version, "]"); j != -1 {
			version = version[:j]
		}
	}
	return version
}
