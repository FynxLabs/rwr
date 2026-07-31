package system

import (
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

// CleanPackageManagers runs the cleanup command for each available package manager
// to free disk space and clear cached package data.
func CleanPackageManagers(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Clean each available package manager
	for name, pm := range osInfo.PackageManager.Managers {
		if pm.Clean == "" {
			continue
		}

		log.Debugf("Running clean command for package manager: %s", name)
		log.Debugf(" Running clean command: %s", pm.Clean)

		// pm.Clean is "<bin> <clean args>"; split it back into argv rather than
		// handing the whole string to a shell.
		cleanCmd := types.Command{
			Exec:     pm.Bin,
			Args:     strings.Fields(strings.TrimPrefix(pm.Clean, pm.Bin)),
			Elevated: pm.Elevated,
		}

		if err := RunCommand(cleanCmd, initConfig.Variables.Flags.Debug); err != nil {
			log.Errorf("Error cleaning package manager %s: %v", name, err)
			continue
		}

		log.Infof("Cleaned package manager: %s", name)
	}

	return nil
}
