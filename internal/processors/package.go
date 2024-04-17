package processors

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/types"
	"os"
	"path/filepath"
)

func ProcessPackagesFromFile(blueprintFile string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var packages []types.Package
	var PackagesData types.PackagesData
	var failedPackages []string

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

	log.Infof("Processing packages from %s", blueprintFile)

	// Install the packages
	for _, pkg := range packages {
		log.Infof("Processing %d packages", len(pkg.Names))
		if len(pkg.Names) > 0 {
			for _, name := range pkg.Names {
				log.Debugf("Processing package %s", name)
				log.Debugf("PackageManager: %s", pkg.PackageManager)
				log.Debugf("Elevated: %t", pkg.Elevated)
				log.Debugf("Action: %s", pkg.Action)
				err := HandlePackage(types.Package{
					Name:           name,
					Elevated:       pkg.Elevated,
					Action:         pkg.Action,
					PackageManager: pkg.PackageManager,
				}, osInfo, initConfig)
				if err != nil {
					failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", name, err))
				}
			}
		} else {
			err := HandlePackage(pkg, osInfo, initConfig)
			if err != nil {
				failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", pkg.Name, err))
			}
		}
	}

	if len(failedPackages) > 0 {
		log.Warnf("Failed to install the following packages:")
		for _, failedPackage := range failedPackages {
			log.Warn(failedPackage)
		}
	}

	return nil
}

func ProcessPackagesFromData(blueprintData []byte, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var packages []types.Package
	var PackagesData types.PackagesData
	var failedPackages []string

	log.Debugf("Processing packages from data")

	// Unmarshal the resolved blueprint data
	log.Debugf("Unmarshaling package blueprint data")
	err := helpers.UnmarshalBlueprint(blueprintData, initConfig.Init.Format, &PackagesData)
	if err != nil {
		log.Errorf("Error unmarshaling package blueprint data: %v", err)
		return fmt.Errorf("error unmarshaling package blueprint data: %w", err)
	}

	packages = PackagesData.Packages

	log.Debugf("Processing %d packages", len(packages))
	log.Debugf("Packages: %v", packages)

	// Install the packages
	for _, pkg := range packages {
		log.Debugf("Processing package(s): %v", pkg.Names)
		if len(pkg.Names) > 0 {
			for _, name := range pkg.Names {
				log.Debugf("Processing package %s", name)
				log.Debugf("PackageManager: %s", pkg.PackageManager)
				log.Debugf("Elevated: %t", pkg.Elevated)
				log.Debugf("Action: %s", pkg.Action)
				err := HandlePackage(types.Package{
					Name:           name,
					Elevated:       pkg.Elevated,
					Action:         pkg.Action,
					PackageManager: pkg.PackageManager,
				}, osInfo, initConfig)
				if err != nil {
					failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", name, err))
				}
			}
		} else {
			err := HandlePackage(pkg, osInfo, initConfig)
			if err != nil {
				failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", pkg.Name, err))
			}
		}
	}

	if len(failedPackages) > 0 {
		log.Warnf("Failed to install the following packages:")
		for _, failedPackage := range failedPackages {
			log.Warn(failedPackage)
		}
	}

	return nil
}

func HandlePackage(pkg types.Package, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var command string
	var install string
	var remove string
	var elevated bool

	if pkg.PackageManager != "" {
		// Use the specified package manager
		switch pkg.PackageManager {
		case "brew":
			log.Debug("Using Homebrew package manager")
			install = osInfo.PackageManager.Brew.Install
			remove = osInfo.PackageManager.Brew.Remove
			elevated = osInfo.PackageManager.Brew.Elevated
		case "apt":
			log.Debug("Using APT package manager")
			install = osInfo.PackageManager.Apt.Install
			remove = osInfo.PackageManager.Apt.Remove
			elevated = osInfo.PackageManager.Apt.Elevated
		case "dnf":
			log.Debug("Using DNF package manager")
			install = osInfo.PackageManager.Dnf.Install
			remove = osInfo.PackageManager.Dnf.Remove
			elevated = osInfo.PackageManager.Dnf.Elevated
		case "eopkg":
			log.Debug("Using Solus eopkg package manager")
			install = osInfo.PackageManager.Eopkg.Install
			remove = osInfo.PackageManager.Eopkg.Remove
			elevated = osInfo.PackageManager.Eopkg.Elevated
		case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
			log.Debugf("Using AUR package manager: %s", pkg.PackageManager)
			install = osInfo.PackageManager.Default.Install
			remove = osInfo.PackageManager.Default.Remove
			elevated = osInfo.PackageManager.Default.Elevated
		case "pacman":
			log.Debug("Using Pacman package manager")
			install = osInfo.PackageManager.Pacman.Install
			remove = osInfo.PackageManager.Pacman.Remove
			elevated = osInfo.PackageManager.Pacman.Elevated
		case "zypper":
			log.Debug("Using Zypper package manager")
			install = osInfo.PackageManager.Zypper.Install
			remove = osInfo.PackageManager.Zypper.Remove
			elevated = osInfo.PackageManager.Zypper.Elevated
		case "emerge":
			log.Debug("Using Gentoo Portage package manager")
			install = osInfo.PackageManager.Emerge.Install
			remove = osInfo.PackageManager.Emerge.Remove
			elevated = osInfo.PackageManager.Emerge.Elevated
		case "nix":
			log.Debug("Using Nix package manager")
			install = osInfo.PackageManager.Nix.Install
			remove = osInfo.PackageManager.Nix.Remove
			elevated = osInfo.PackageManager.Nix.Elevated
		case "cargo":
			log.Debug("Using Cargo package manager")
			install = osInfo.PackageManager.Cargo.Install
			remove = osInfo.PackageManager.Cargo.Remove
			elevated = osInfo.PackageManager.Cargo.Elevated
		default:
			return fmt.Errorf("unsupported package manager: %s", pkg.PackageManager)
		}
	} else {
		log.Debugf("Using default package manager: %s", osInfo.PackageManager.Default.Name)
		// Use the default package manager
		install = osInfo.PackageManager.Default.Install
		remove = osInfo.PackageManager.Default.Remove
		elevated = osInfo.PackageManager.Default.Elevated
	}

	// Override the elevated flag if specified in the package configuration
	if pkg.Elevated {
		elevated = true
	}

	var args []string
	args = append(args, pkg.Name)

	if pkg.Action == "install" {
		log.Debugf("Installing package %s", pkg.Name)
		command = install
	} else if pkg.Action == "remove" {
		log.Debugf("Removing package %s", pkg.Name)
		command = remove
	} else {
		return fmt.Errorf("unsupported action: %s", pkg.Action)
	}

	pkgCmd := types.Command{
		Exec:     command,
		Args:     args,
		Elevated: elevated,
	}

	err := helpers.RunCommand(pkgCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error processing package %s: %v", pkg.Name, err)
	}

	return nil
}
