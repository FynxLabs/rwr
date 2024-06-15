package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// SetMacOSDetails Sets the package manager details for macOS.
func SetMacOSDetails(osInfo *types.OSInfo) {
	log.Debug("Setting macOS package manager details.")

	//TODO: Move all package manager actions to a separate file to avoid duplication

	if CommandExists("brew") {
		log.Debug("Homebrew detected.")
		osInfo.PackageManager.Brew = types.PackageManagerInfo{
			Bin:      "brew",
			List:     "brew list",
			Search:   "brew search",
			Install:  "brew install -fq",
			Remove:   "brew uninstall -fq",
			Update:   "brew update && brew upgrade",
			Clean:    "brew cleanup -q",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Brew
	}

	if CommandExists("nix-env") {
		log.Debug("Nix detected.")
		osInfo.PackageManager.Nix = types.PackageManagerInfo{
			Bin:      "nix-env",
			List:     "nix-env -q",
			Search:   "nix search",
			Install:  "nix-env -i",
			Remove:   "nix-env -e",
			Update:   "nix-channel --update && nix-env -u '*'",
			Clean:    "nix-collect-garbage -d",
			Elevated: false,
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		}
	}

	if CommandExists("mas") {
		log.Debug("Mac App Store CLI detected.")
		osInfo.PackageManager.MAS = types.PackageManagerInfo{
			Bin:      "mas",
			List:     "mas list",
			Search:   "mas search",
			Install:  "mas install",
			Update:   "mas upgrade",
			Remove:   "mas uninstall",
			Clean:    "mas reset",
			Elevated: false,
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.MAS
		}
	}

	if CommandExists("cargo") {
		log.Debug("Cargo detected.")
		osInfo.PackageManager.Cargo = types.PackageManagerInfo{
			Bin:      "cargo",
			List:     "cargo install --list",
			Search:   "cargo search",
			Install:  "cargo install",
			Remove:   "cargo uninstall",
			Update:   "cargo install --force",
			Clean:    "cargo cache --autoclean",
			Elevated: false,
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Cargo
		}
	}

	if CommandExists("port") {
		log.Debug("MacPorts detected.")
		osInfo.PackageManager.MacPorts = types.PackageManagerInfo{
			Bin:      "port",
			List:     "port installed",
			Search:   "port search",
			Install:  "port install",
			Remove:   "port uninstall",
			Update:   "port selfupdate && port upgrade outdated",
			Clean:    "port clean --all all",
			Elevated: true,
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.MacPorts
		}
	}

	// Override default package manager if set in viper config
	viperDefault := viper.GetString("packageManager.macos.default")
	if viperDefault != "" {
		log.Debugf("Overriding default package manager with value from Viper: %s", viperDefault)
		switch viperDefault {
		case "brew":
			osInfo.PackageManager.Default = osInfo.PackageManager.Brew
		case "nix":
			osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		case "mas":
			osInfo.PackageManager.Default = osInfo.PackageManager.MAS
		case "cargo":
			osInfo.PackageManager.Default = osInfo.PackageManager.Cargo
		case "port":
			osInfo.PackageManager.Default = osInfo.PackageManager.MacPorts
		default:
			log.Warnf("Unknown default package manager specified in Viper config: %s", viperDefault)
		}
	}
}
