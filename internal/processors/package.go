package processors

import (
	"encoding/json"
	"fmt"
	"github.com/thefynx/rwr/internal/helpers"
	"github.com/thefynx/rwr/internal/processors/types"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func ProcessPackages(manifestPath string, osInfo types.OSInfo) error {
	var config types.Config

	ext := filepath.Ext(manifestPath)
	switch ext {
	case ".toml":
		_, err := toml.DecodeFile(manifestPath, &config)
		if err != nil {
			return fmt.Errorf("error decoding TOML file: %v", err)
		}
	case ".yaml", ".yml":
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("error reading YAML file: %v", err)
		}
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("error decoding YAML file: %v", err)
		}
	case ".json":
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("error reading JSON file: %v", err)
		}
		err = json.Unmarshal(data, &config)
		if err != nil {
			return fmt.Errorf("error decoding JSON file: %v", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	for _, pkg := range config.Packages {
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

		var names []string
		if pkg.Name != "" {
			names = []string{pkg.Name}
		} else {
			names = pkg.Names
		}

		for _, name := range names {
			var args []string
			if pkg.Action == "install" {
				args = append(args, "install", name)
			} else if pkg.Action == "remove" {
				args = append(args, "remove", name)
			} else {
				return fmt.Errorf("unsupported action: %s", pkg.Action)
			}

			if elevated {
				err := helpers.RunWithElevatedPrivileges(command, args...)
				if err != nil {
					return fmt.Errorf("error processing package %s: %v", name, err)
				}
			} else {
				err := helpers.RunCommand(command, args...)
				if err != nil {
					return fmt.Errorf("error processing package %s: %v", name, err)
				}
			}
		}
	}

	return nil
}
