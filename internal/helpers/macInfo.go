package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/processors/types"
)

// SetMacOSDetails Sets the package manager details for macOS.
func SetMacOSDetails(osInfo *types.OSInfo) {
	log.Debug("Setting macOS package manager details.")

	if CommandExists("brew") {
		log.Debug("Homebrew detected.")
		osInfo.PackageManager.Brew = types.PackageManagerInfo{
			Bin:     "brew",
			List:    "brew list",
			Search:  "brew search",
			Install: "brew install -fq",
			Update:  "brew update && brew upgrade",
			Clean:   "brew cleanup -q",
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Brew
	}

	if CommandExists("nix-env") {
		log.Debug("Nix detected.")
		osInfo.PackageManager.Nix = types.PackageManagerInfo{
			Bin:     "nix-env",
			List:    "nix-env -q",
			Search:  "nix search",
			Install: "nix-env -i",
			Update:  "nix-channel --update && nix-env -u '*'",
			Clean:   "nix-collect-garbage -d",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		}
	}

	if CommandExists("mas") {
		log.Debug("Mac App Store CLI detected.")
		osInfo.PackageManager.MAS = types.PackageManagerInfo{
			Bin:     "mas",
			List:    "mas list",
			Search:  "mas search",
			Install: "mas install",
			Update:  "mas upgrade",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.MAS
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
		default:
			log.Warnf("Unknown default package manager specified in Viper config: %s", viperDefault)
		}
	}
}
