package types

import "fmt"

// ValidatePackageManagers checks all declarations before any install can run.
func ValidatePackageManagers(managers []PackageManagerInfo) error {
	for i, manager := range managers {
		if manager.Name == "" {
			return fmt.Errorf("missing required field 'packageManagers[%d].name'", i)
		}
		if manager.Action != "install" && manager.Action != "remove" {
			return fmt.Errorf("packageManagers[%d].action must be install or remove", i)
		}
	}
	return nil
}
