package helpers

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
)

func CleanPackageManagers(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	packageManagers := getPackageManagerNames(osInfo.PackageManager)

	for _, pm := range packageManagers {
		if pm == "default" {
			continue
		}

		pmInfo, err := GetPackageManagerInfo(osInfo, pm)
		if err != nil {
			log.Debugf("Package manager not found: %s", pm)
			continue
		}

		if CommandExists(pmInfo.Bin) {
			log.Debugf("Running clean command for package manager: %s", pm)
			log.Debugf(" Running clean command: %s", pmInfo.Clean)
			err = cleanPackageManager(pmInfo, initConfig)
			if err != nil {
				log.Errorf("Error cleaning package manager %s: %v", pm, err)
				return err
			}
			log.Infof("Cleaned package manager: %s", pm)
		} else {
			log.Debugf("Package manager not found: %s", pm)
		}
	}

	return nil
}

func cleanPackageManager(pmInfo types.PackageManagerInfo, initConfig *types.InitConfig) error {

	cleanCmd := types.Command{
		Exec:     pmInfo.Clean,
		Elevated: pmInfo.Elevated,
	}

	err := RunCommand(cleanCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error running clean command: %v", err)
	}

	return nil
}
