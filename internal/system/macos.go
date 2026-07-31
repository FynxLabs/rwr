package system

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// SetMacOSDetails populates the OSInfo struct with macOS-specific package manager
// details, detecting Homebrew and other available providers.
func SetMacOSDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting macOS package manager details.")

	collectAvailableManagers(osInfo)
	setDefaultManager(osInfo, []string{"brew", "macports"})

	return nil
}

// getDarwinVersion returns the macOS version.
func getDarwinVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	out, err := cmd.Output()
	if err != nil {
		log.Warnf("Error getting macOS version: %v", err)
		return "Unknown"
	}
	return strings.TrimSpace(string(out))
}
