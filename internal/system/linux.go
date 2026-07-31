package system

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
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
// this system: whatever /etc/os-release maps to first, then — on Arch-family
// distributions — the AUR helpers ahead of bare pacman.
func linuxPreferredManagers(osInfo *types.OSInfo) []string {
	var preferred []string

	if fromOSRelease := GetDefaultProviderFromOSRelease(); fromOSRelease != "" {
		preferred = append(preferred, fromOSRelease)
	}

	if IsDistroInFamily(osInfo.System.OSFamily, "arch") {
		preferred = append(preferred, "paru", "yay", "trizen", "aura", "pamac", "pacman")
	}

	return preferred
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
