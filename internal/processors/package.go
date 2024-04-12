package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"path/filepath"
)

func ProcessPackagesFromFile(blueprintFile string, osInfo types.OSInfo) error {
	var packages []types.Package
	var PackagesData types.PackagesData

	// Read the blueprint file
	log.Debugf("Reading blueprint file %s", blueprintFile)
	blueprintData, err := os.ReadFile(blueprintFile)
	if err != nil {
		return fmt.Errorf("error reading blueprint file: %w", err)
	}

	// Unmarshal the blueprint data
	log.Debugf("Unmarshaling blueprint data from %s", blueprintFile)
	err = helpers.UnmarshalBlueprint(blueprintData, filepath.Ext(blueprintFile), &PackagesData)
	if err != nil {
		return fmt.Errorf("error unmarshaling package blueprint: %w", err)
	}

	packages = PackagesData.Packages

	log.Debugf("Processing %d packages", len(packages))

	// Install the packages
	for _, pkg := range packages {
		if len(pkg.Names) > 0 {
			for _, name := range pkg.Names {
				log.Debugf("Processing package %s", name)
				log.Debugf("PackageManager: %s", pkg.PackageManager)
				log.Debugf("Elevated: %t", pkg.Elevated)
				log.Debugf("Action: %s", pkg.Action)
				err := InstallPackage(types.Package{
					Name:           name,
					Elevated:       pkg.Elevated,
					Action:         pkg.Action,
					PackageManager: pkg.PackageManager,
				}, osInfo)
				if err != nil {
					return fmt.Errorf("error installing package %s: %w", name, err)
				}
			}
		} else {
			err := InstallPackage(pkg, osInfo)
			if err != nil {
				return fmt.Errorf("error installing package %s: %w", pkg.Name, err)
			}
		}
	}

	return nil
}

func ProcessPackagesFromData(blueprintData []byte, initConfig *types.InitConfig, osInfo types.OSInfo) error {
	var packages []types.Package
	var PackagesData types.PackagesData

	log.Debugf("Processing packages from data")

	// Unmarshal the resolved blueprint data
	log.Debugf("Unmarshaling package blueprint data")
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &PackagesData)
	if err != nil {
		return fmt.Errorf("error unmarshaling package blueprint data: %w", err)
	}

	packages = PackagesData.Packages

	log.Debugf("Processing %d packages", len(packages))
	// Install the packages
	for _, pkg := range packages {
		if len(pkg.Names) > 0 {
			for _, name := range pkg.Names {
				log.Debugf("Processing package %s", name)
				log.Debugf("PackageManager: %s", pkg.PackageManager)
				log.Debugf("Elevated: %t", pkg.Elevated)
				log.Debugf("Action: %s", pkg.Action)
				err := InstallPackage(types.Package{
					Name:           name,
					Elevated:       pkg.Elevated,
					Action:         pkg.Action,
					PackageManager: pkg.PackageManager,
				}, osInfo)
				if err != nil {
					return fmt.Errorf("error installing package %s: %w", name, err)
				}
			}
		} else {
			err := InstallPackage(pkg, osInfo)
			if err != nil {
				return fmt.Errorf("error installing package %s: %w", pkg.Name, err)
			}
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
			log.Debug("Using Homebrew package manager")
			command = osInfo.PackageManager.Brew.Bin
			elevated = osInfo.PackageManager.Brew.Elevated
		case "apt":
			log.Debug("Using APT package manager")
			command = osInfo.PackageManager.Apt.Bin
			elevated = osInfo.PackageManager.Apt.Elevated
		case "dnf":
			log.Debug("Using DNF package manager")
			command = osInfo.PackageManager.Dnf.Bin
			elevated = osInfo.PackageManager.Dnf.Elevated
		case "eopkg":
			log.Debug("Using Solus eopkg package manager")
			command = osInfo.PackageManager.Eopkg.Bin
			elevated = osInfo.PackageManager.Eopkg.Elevated
		case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
			log.Debugf("Using AUR package manager: %s", pkg.PackageManager)
			command = osInfo.PackageManager.Default.Bin
			elevated = osInfo.PackageManager.Default.Elevated
		case "pacman":
			log.Debug("Using Pacman package manager")
			command = osInfo.PackageManager.Pacman.Bin
			elevated = osInfo.PackageManager.Pacman.Elevated
		case "zypper":
			log.Debug("Using Zypper package manager")
			command = osInfo.PackageManager.Zypper.Bin
			elevated = osInfo.PackageManager.Zypper.Elevated
		case "emerge":
			log.Debug("Using Gentoo Portage package manager")
			command = osInfo.PackageManager.Emerge.Bin
			elevated = osInfo.PackageManager.Emerge.Elevated
		case "nix":
			log.Debug("Using Nix package manager")
			command = osInfo.PackageManager.Nix.Bin
			elevated = osInfo.PackageManager.Nix.Elevated
		case "cargo":
			log.Debug("Using Cargo package manager")
			command = osInfo.PackageManager.Cargo.Bin
			elevated = osInfo.PackageManager.Cargo.Elevated
		default:
			return fmt.Errorf("unsupported package manager: %s", pkg.PackageManager)
		}
	} else {
		log.Debugf("Using default package manager: %s", osInfo.PackageManager.Default.Name)
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
		err := helpers.RunWithElevatedPrivileges(command, "", args...)
		if err != nil {
			return fmt.Errorf("error processing package %s: %v", pkg.Name, err)
		}
	} else {
		err := helpers.RunCommand(command, "", args...)
		if err != nil {
			return fmt.Errorf("error processing package %s: %v", pkg.Name, err)
		}
	}

	return nil
}
