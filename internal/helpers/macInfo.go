package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

// Sets the package manager details for macOS.
func setMacOSDetails(osInfo *OSInfo) {
	log.Debug("Setting macOS package manager details.")

	if commandExists("brew") {
		log.Debug("Homebrew detected.")
		osInfo.PackageManager.Brew = PackageManagerInfo{
			Bin:     "brew",
			List:    "brew list",
			Search:  "brew search",
			Install: "brew install -fq",
			Clean:   "brew cleanup -q",
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Brew
	}

	if commandExists("nix-env") {
		log.Debug("Nix detected.")
		osInfo.PackageManager.Nix = PackageManagerInfo{
			Bin:     "nix-env",
			List:    "nix-env -q",
			Search:  "nix search",
			Install: "nix-env -i",
			Clean:   "nix-collect-garbage -d",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		}
	}

	if commandExists("mas") {
		log.Debug("Mac App Store CLI detected.")
		osInfo.PackageManager.MAS = PackageManagerInfo{
			Bin:     "mas",
			List:    "mas list",
			Search:  "mas search",
			Install: "mas install",
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
