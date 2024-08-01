package processors

import (
	"bufio"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

func ProcessPackages(blueprintData []byte, packagesData *types.PackagesData, format string, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var failedPackages []string
	var err error

	log.Debugf("Processing packages from blueprint")

	if packagesData == nil {
		// Unmarshal the blueprint data
		err = helpers.UnmarshalBlueprint(blueprintData, format, &packagesData)
		if err != nil {
			return fmt.Errorf("error unmarshaling package blueprint: %w", err)
		}
	}
	log.Debugf("Processing %d packages", len(packagesData.Packages))

	err = helpers.SetPaths()
	if err != nil {
		return fmt.Errorf("error setting paths: %w", err)
	}

	installedPackages := make(map[string]bool)

	// Iterate over all fields in the PackageManager struct
	v := reflect.ValueOf(osInfo.PackageManager)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeOf(types.PackageManagerInfo{}) {
			pm := field.Interface().(types.PackageManagerInfo)
			log.Debugf("Checking for installed packages via: %s", pm.Name)
			if pm.Bin != "" {
				log.Debugf("%s installed, checking packages", pm.Name)
				installed, err := getInstalledPackages(pm)
				if err != nil {
					log.Warnf("Error getting installed packages for %s: %v", pm.Name, err)
				}
				for _, pkg := range installed {
					installedPackages[pkg] = true
				}
			}
		}
	}

	for _, pkg := range packagesData.Packages {
		if len(pkg.Names) > 0 {
			log.Infof("Processing %d packages for %s", len(pkg.Names), pkg.PackageManager)
			for _, name := range pkg.Names {
				if pkg.Action == "install" && installedPackages[name] {
					log.Infof("Package %s is already installed, skipping", name)
					continue
				}

				// This is where the new code snippet goes
				if pkg.PackageManager == "gnome-extensions" {
					extID, err := getGnomeExtensionID(name)
					if err != nil {
						failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", name, err))
						continue
					}
					name = extID
				}

				err := ProcessPackage(types.Package{
					Name:           name,
					Elevated:       pkg.Elevated,
					Action:         pkg.Action,
					PackageManager: pkg.PackageManager,
					Args:           pkg.Args,
				}, osInfo, initConfig)
				if err != nil {
					failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", name, err))
				}
			}
		} else {
			if pkg.Action == "install" && installedPackages[pkg.Name] {
				log.Infof("Package %s is already installed, skipping", pkg.Name)
				continue
			}

			// This is where a similar check would go for single packages
			if pkg.PackageManager == "gnome-extensions" {
				extID, err := getGnomeExtensionID(pkg.Name)
				if err != nil {
					failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", pkg.Name, err))
					continue
				}
				pkg.Name = extID
			}

			err := ProcessPackage(pkg, osInfo, initConfig)
			if err != nil {
				failedPackages = append(failedPackages, fmt.Sprintf("Package %s: %v", pkg.Name, err))
			}
		}
	}

	if len(failedPackages) > 0 {
		log.Warnf("Failed to process the following packages:")
		for _, failedPackage := range failedPackages {
			log.Warn(failedPackage)
		}
	}

	return nil
}

func ProcessPackage(pkg types.Package, osInfo *types.OSInfo, initConfig *types.InitConfig) error {
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
		case "gnome-extensions":
			log.Debug("Using GNOME Extensions CLI package manager")
			install = osInfo.PackageManager.GnomeExtensions.Install
			remove = osInfo.PackageManager.GnomeExtensions.Remove
			elevated = osInfo.PackageManager.GnomeExtensions.Elevated
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

	// Add any additional arguments specified in the package configuration
	args = append(args, pkg.Args...)

	if pkg.Action == "install" {
		log.Debugf("Installing package %s", pkg.Name)
		command = install
		args = append(args, pkg.Name)
	} else if pkg.Action == "remove" {
		log.Debugf("Removing package %s", pkg.Name)
		command = remove
		args = append(args, pkg.Name)
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

func getInstalledPackages(pm types.PackageManagerInfo) ([]string, error) {
	listCmd := types.Command{
		Exec: pm.List,
	}

	output, err := helpers.RunCommandOutput(listCmd, false)
	if err != nil {
		return nil, fmt.Errorf("error getting installed packages: %v", err)
	}

	if pm.Name == "gnome-extensions" {
		var packages []string
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) > 0 {
				packages = append(packages, fields[0])
			}
		}
		return packages, nil
	}

	return strings.Fields(output), nil
}

func getGnomeExtensionID(nameOrID string) (string, error) {
	if _, err := strconv.Atoi(nameOrID); err == nil {
		return nameOrID, nil
	}

	searchCmd := types.Command{
		Exec: "gext",
		Args: []string{"search", nameOrID},
	}

	output, err := helpers.RunCommandOutput(searchCmd, false)
	if err != nil {
		return "", fmt.Errorf("error searching for GNOME extension: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && strings.Contains(strings.ToLower(scanner.Text()), strings.ToLower(nameOrID)) {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("GNOME extension not found: %s", nameOrID)
}
