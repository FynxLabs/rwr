package system

import (
	"os"
	"sort"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetLinuxDetails populates the OSInfo struct with Linux-specific package manager
// details by querying available providers and detecting the default package manager.
func SetLinuxDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Linux package manager details.")

	collectAvailableManagers(osInfo)
	setDefaultManager(osInfo, linuxPreferredManagers(osInfo))

	if osInfo.PackageManager.Default.Name != "" {
		log.Infof("Final default package manager: %s", osInfo.PackageManager.Default.Name)
	}

	return nil
}

// linuxPreferredManagers returns the default package manager preference order for
// this system: the distribution family's own package managers, AUR helpers ahead
// of bare pacman on Arch.
//
// Deriving the default from /etc/os-release directly does not work. That lookup
// returned the first provider matching the distro, and flatpak, snap, nix and
// cargo all declare the "linux" wildcard, so they match every distribution — on
// this machine it selected flatpak as the default package manager for an
// Arch-based system. Families map to their native managers explicitly instead.
func linuxPreferredManagers(osInfo *types.OSInfo) []string {
	family := linuxFamily(osInfo)
	if family == "" {
		log.Debug("Could not determine distribution family; falling back to alphabetical default")
		return nil
	}

	log.Debugf("Distribution family resolved to %q", family)
	return nativeManagers[family]
}

// linuxFamily determines which distribution family this machine belongs to,
// preferring what /etc/os-release reports and falling back to what is installed.
//
// The fallback matters because derivative distributions routinely name themselves
// something new and set no ID_LIKE, leaving nothing to match on. A machine with
// pacman and /var/lib/pacman is Arch-family regardless of what it calls itself.
func linuxFamily(osInfo *types.OSInfo) string {
	if family := GetDistroFamily(osInfo.System.OSFamily); family != "" {
		if _, known := nativeManagers[family]; known {
			return family
		}
	}

	// Infer from the package managers actually present. Sorted for determinism,
	// though in practice a system carries only one native package manager.
	installed := make([]string, 0, len(osInfo.PackageManager.Managers))
	for name := range osInfo.PackageManager.Managers {
		installed = append(installed, name)
	}
	sort.Strings(installed)

	for _, name := range installed {
		if family, ok := managerFamily[name]; ok {
			log.Debugf("Distribution %q is unrecognised; inferred %q family from the presence of %s",
				osInfo.System.OSFamily, family, name)
			return family
		}
	}

	return ""
}

// getLinuxDistro returns the Linux distribution name from /etc/os-release.
func getLinuxDistro() string {
	log.Debug("Starting Linux distribution detection")

	// Try /etc/os-release first (standard location)
	if fileExists("/etc/os-release") {
		log.Debug("Found /etc/os-release, reading distribution ID")
		content, err := os.ReadFile("/etc/os-release")
		if err != nil {
			log.Debugf("Error reading /etc/os-release: %v", err)
		} else {
			log.Debugf("Successfully read /etc/os-release (%d bytes)", len(content))
			lines := strings.Split(string(content), "\n")
			log.Debugf("Parsing %d lines from /etc/os-release", len(lines))

			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				log.Debugf("Line %d: %s", i+1, line)
				if strings.HasPrefix(line, "ID=") {
					id := strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
					log.Debugf("Successfully extracted Linux ID from /etc/os-release: '%s'", id)
					return id
				}
			}
			log.Debug("No ID= field found in /etc/os-release")
		}
	} else {
		log.Debug("/etc/os-release does not exist")
	}

	// Fallback to /etc/lsb-release
	if fileExists("/etc/lsb-release") {
		log.Debug("Found /etc/lsb-release, reading distribution ID as fallback")
		content, err := os.ReadFile("/etc/lsb-release")
		if err != nil {
			log.Debugf("Error reading /etc/lsb-release: %v", err)
		} else {
			log.Debugf("Successfully read /etc/lsb-release (%d bytes)", len(content))
			lines := strings.Split(string(content), "\n")
			log.Debugf("Parsing %d lines from /etc/lsb-release", len(lines))

			for i, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				log.Debugf("Line %d: %s", i+1, line)
				if strings.HasPrefix(line, "DISTRIB_ID=") {
					id := strings.Trim(strings.TrimPrefix(line, "DISTRIB_ID="), "\"")
					log.Debugf("Successfully extracted Linux ID from /etc/lsb-release: '%s'", id)
					return id
				}
			}
			log.Debug("No DISTRIB_ID= field found in /etc/lsb-release")
		}
	} else {
		log.Debug("/etc/lsb-release does not exist")
	}

	log.Warn("Failed to detect Linux distribution from both /etc/os-release and /etc/lsb-release, returning 'Unknown Linux'")
	return "Unknown Linux"
}

// getLinuxVersion returns the Linux version from /etc/os-release.
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

// fileExists checks if a file exists and is not a directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
