package system

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetLinuxDetails sets Linux-specific system details
func SetLinuxDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Linux package manager details.")

	// Initialize package manager map
	if osInfo.PackageManager.Managers == nil {
		osInfo.PackageManager.Managers = make(map[string]types.PackageManagerInfo)
	}

	// Get available providers
	available := GetAvailableProviders()

	// Add all available Linux package managers
	for name, prov := range available {
		// Skip if not a Linux provider
		if !Contains(prov.Detection.Distributions, "linux") {
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

	// Get default from OS release
	defaultPackageManager := GetDefaultProviderFromOSRelease()
	if defaultPackageManager != "" && osInfo.PackageManager.Managers[defaultPackageManager].Bin != "" {
		osInfo.PackageManager.Default = osInfo.PackageManager.Managers[defaultPackageManager]
		log.Infof("Set %s as default package manager from OS release", defaultPackageManager)
	} else {
		// Check if this is an Arch-based distribution
		if IsDistroInFamily(osInfo.System.OSFamily, "arch") {
			// For Arch Linux, prioritize AUR helpers (in order of preference)
			aurHelpers := []string{"paru", "yay", "trizen", "aura", "pamac"}
			for _, helper := range aurHelpers {
				if pm, exists := osInfo.PackageManager.Managers[helper]; exists && pm.Bin != "" {
					osInfo.PackageManager.Default = pm
					log.Infof("Set AUR helper %s as default package manager for Arch-based system", helper)
					break
				}
			}

			// If no AUR helper found, try pacman
			if osInfo.PackageManager.Default.Name == "" {
				if pm, exists := osInfo.PackageManager.Managers["pacman"]; exists && pm.Bin != "" {
					osInfo.PackageManager.Default = pm
					log.Infof("Set pacman as default package manager for Arch-based system")
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
	}

	// Debug log the final default package manager
	if osInfo.PackageManager.Default.Name != "" {
		log.Infof("Final default package manager: %s", osInfo.PackageManager.Default.Name)
	} else {
		log.Warn("No default package manager set")
	}

	return nil
}

// getLinuxDistro returns the Linux distribution name from /etc/os-release
func getLinuxDistro() string {
	if fileExists("/etc/os-release") {
		log.Debugf("Getting Linux Dristro ID from /etc/os-release")
		content, err := os.ReadFile("/etc/os-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "ID=") {
					id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
					log.Debugf("Found Linux ID: %s", id)
					return id
				}
			}
		}
	}

	if fileExists("/etc/lsb-release") {
		log.Debugf("Getting Linux Dristro ID from /etc/lsb-release")
		content, err := os.ReadFile("/etc/lsb-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "DISTRIB_ID=") {
					id := strings.Trim(strings.TrimPrefix(line, "DISTRIB_ID="), "\"")
					log.Debugf("Found Linux ID: %s", id)
					return id
				}
			}
		}
	}

	return "Unknown Linux"
}

// getLinuxVersion returns the Linux version from /etc/os-release
func getLinuxVersion() string {
	if fileExists("/etc/os-release") {
		log.Debugf("Getting Linux Version from /etc/os-release")
		content, err := os.ReadFile("/etc/os-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "VERSION_ID=") {
					version := strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
					log.Debugf("Found Linux Version: %s", version)
					return version
				}
			}
		}
	}

	if fileExists("/etc/lsb-release") {
		log.Debugf("Getting Linux Version from /etc/lsb-release")
		content, err := os.ReadFile("/etc/lsb-release")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "DISTRIB_RELEASE=") {
					version := strings.Trim(strings.TrimPrefix(line, "DISTRIB_RELEASE="), "\"")
					log.Debugf("Found Linux Version: %s", version)
					return version
				}
			}
		}
	}

	return "Unknown Version"
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Contains checks if a string slice contains a string
func Contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
