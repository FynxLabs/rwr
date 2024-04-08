package processors

import (
	"fmt"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
)

func ProcessPackagesFromFile(blueprintFile string, osInfo types.OSInfo) error {
	var packages []types.Package

	// Read the blueprint file
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &packages)
	if err != nil {
		return fmt.Errorf("error unmarshaling package blueprint: %w", err)
	}

	// Install the packages
	for _, pkg := range packages {
		err := InstallPackage(pkg, osInfo)
		if err != nil {
			return fmt.Errorf("error installing package %s: %w", pkg.Name, err)
		}
	}

	return nil
}

func ProcessPackagesFromData(blueprintData []byte, initConfig *types.InitConfig, osInfo types.OSInfo) error {
	var packages []types.Package

	// Unmarshal the resolved blueprint data
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Blueprint.Format, &packages)
	if err != nil {
		return fmt.Errorf("error unmarshaling package blueprint data: %w", err)
	}

	// Install the packages
	for _, pkg := range packages {
		err := InstallPackage(pkg, osInfo)
		if err != nil {
			return fmt.Errorf("error installing package %s: %w", pkg.Name, err)
		}
	}

	return nil
}
func InstallPackage(pkg types.Package, osInfo types.OSInfo) error {
	var command string
	var elevated bool

	if pkg.PackageManager != "" {
		// Use the specified package manager
		switch pkg.PackageManager {
		case "brew":
			command = osInfo.PackageManager.Brew.Bin
			elevated = osInfo.PackageManager.Brew.Elevated
		case "apt":
			command = osInfo.PackageManager.Apt.Bin
			elevated = osInfo.PackageManager.Apt.Elevated
		case "dnf":
			command = osInfo.PackageManager.Dnf.Bin
			elevated = osInfo.PackageManager.Dnf.Elevated
		case "eopkg":
			command = osInfo.PackageManager.Eopkg.Bin
			elevated = osInfo.PackageManager.Eopkg.Elevated
		case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
			command = osInfo.PackageManager.Default.Bin
			elevated = osInfo.PackageManager.Default.Elevated
		case "pacman":
			command = osInfo.PackageManager.Pacman.Bin
			elevated = osInfo.PackageManager.Pacman.Elevated
		case "zypper":
			command = osInfo.PackageManager.Zypper.Bin
			elevated = osInfo.PackageManager.Zypper.Elevated
		case "emerge":
			command = osInfo.PackageManager.Emerge.Bin
			elevated = osInfo.PackageManager.Emerge.Elevated
		case "nix":
			command = osInfo.PackageManager.Nix.Bin
			elevated = osInfo.PackageManager.Nix.Elevated
		default:
			return fmt.Errorf("unsupported package manager: %s", pkg.PackageManager)
		}
	} else {
		// Use the default package manager
		command = osInfo.PackageManager.Default.Bin
		elevated = osInfo.PackageManager.Default.Elevated
	}

	// Override the elevated flag if specified in the package configuration
	if pkg.Elevated {
		elevated = true
	}

	var args []string
	if pkg.Action == "install" {
		args = append(args, "install", pkg.Name)
	} else if pkg.Action == "remove" {
		args = append(args, "remove", pkg.Name)
	} else {
		return fmt.Errorf("unsupported action: %s", pkg.Action)
	}

	if elevated {
		err := helpers.RunWithElevatedPrivileges(command, args...)
		if err != nil {
			return fmt.Errorf("error processing package %s: %v", pkg.Name, err)
		}
	} else {
		err := helpers.RunCommand(command, args...)
		if err != nil {
			return fmt.Errorf("error processing package %s: %v", pkg.Name, err)
		}
	}

	return nil
}
