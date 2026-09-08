package types

import (
	"errors"
	"fmt"
)

// ErrInvalidPackageManager identifies an invalid package-manager declaration.
var ErrInvalidPackageManager = errors.New("invalid package-manager configuration")

// ValidatePackageManagers checks all declarations before any install can run.
func ValidatePackageManagers(managers []PackageManagerInfo) error {
	for i, manager := range managers {
		if manager.Name == "" {
			return fmt.Errorf("%w: missing required field 'packageManagers[%d].name'", ErrInvalidPackageManager, i)
		}
		if manager.Action != "install" && manager.Action != "remove" {
			return fmt.Errorf("%w: packageManagers[%d].action must be install or remove", ErrInvalidPackageManager, i)
		}
	}
	return nil
}
