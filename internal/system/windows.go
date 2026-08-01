package system

import (
	"os/exec"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetWindowsDetails populates the OSInfo struct with Windows-specific package manager
// details, detecting Chocolatey, Scoop, and WinGet providers.
func SetWindowsDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Windows package manager details.")

	collectAvailableManagers(osInfo)
	setDefaultManager(osInfo, []string{"winget", "chocolatey", "scoop"})

	return nil
}

// getWindowsVersion returns the Windows version.
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
