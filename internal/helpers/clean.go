package helpers

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/processors/types"
)

func CleanPackageManagers(osInfo *types.OSInfo) error {
	packageManagers := getPackageManagerNames(osInfo.PackageManager)

	for _, pm := range packageManagers {
		pmInfo, err := GetPackageManagerInfo(osInfo, pm)
		if err != nil {
			log.Debugf("Package manager not found: %s", pm)
			continue
		}

		log.Debugf("Running clean command for package manager: %s", pm)
		err = cleanPackageManager(pmInfo)
		if err != nil {
			log.Errorf("Error cleaning package manager %s: %v", pm, err)
			return err
		}
		log.Infof("Cleaned package manager: %s", pm)
	}

	return nil
}

func cleanPackageManager(pmInfo types.PackageManagerInfo) error {
	clean := pmInfo.Clean
	elevated := pmInfo.Elevated

	if elevated {
		err := RunWithElevatedPrivileges(clean, "")
		if err != nil {
			return fmt.Errorf("error running clean command with elevated privileges: %v", err)
		}
	} else {
		err := RunCommand(clean, "")
		if err != nil {
			return fmt.Errorf("error running clean command: %v", err)
		}
	}

	return nil
}
