package system

import (
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

func CleanPackageManagers(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	// Clean each available package manager
	for name, pm := range osInfo.PackageManager.Managers {
		if pm.Clean == "" {
			continue
		}

		log.Debugf("Running clean command for package manager: %s", name)
		log.Debugf(" Running clean command: %s", pm.Clean)

		cleanCmd := types.Command{
			Exec:     pm.Clean,
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
