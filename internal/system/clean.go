package system

import (
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// CleanPackageManagers runs the cleanup command for each available package manager
// to free disk space and clear cached package data.
func CleanPackageManagers(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Clean each available package manager
	for name, pm := range osInfo.PackageManager.Managers {
		// GetPackageManagerInfo builds Clean as "<bin> <clean args>", so a provider
		// that defines no clean command still yields "<bin> " - never "". This guard
		// therefore never fired, and the bare provider binary was executed at the end
		// of every run. For an AUR helper, a bare invocation can mean a full system
		// upgrade. Compare the arguments, not the concatenation.
		cleanArgs := strings.Fields(strings.TrimPrefix(pm.Clean, pm.Bin))
		if len(cleanArgs) == 0 {
			log.Debugf("Package manager %s defines no clean command, skipping", name)
			continue
		}

		log.Debugf("Running clean command for package manager: %s", name)
		log.Debugf(" Running clean command: %s", pm.Clean)

		cleanCmd := types.Command{
			Exec:     pm.Bin,
			Args:     cleanArgs,
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
