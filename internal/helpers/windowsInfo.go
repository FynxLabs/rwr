package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/processors/types"
)

// SetWindowsDetails Sets the package manager details for Windows.
func SetWindowsDetails(osInfo *types.OSInfo) {
	log.Debug("Setting Windows package manager details.")

	if CommandExists("choco") {
		log.Debug("Chocolatey detected.")
		osInfo.PackageManager.Chocolatey = types.PackageManagerInfo{
			Bin:     "choco",
			List:    "choco list --local-only",
			Search:  "choco search",
			Install: "choco install -y",
			Update:  "choco upgrade -y all",
			Clean:   "choco cache delete",
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Chocolatey
	}

	if CommandExists("scoop") {
		log.Debug("Scoop detected.")
		osInfo.PackageManager.Scoop = types.PackageManagerInfo{
			Bin:     "scoop",
			List:    "scoop list",
			Search:  "scoop search",
			Install: "scoop install",
			Update:  "scoop update",
			Clean:   "scoop cache rm *",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Scoop
		}
	}

	// Override default package manager if set in viper config
	viperDefault := viper.GetString("packageManager.windows.default")
	if viperDefault != "" {
		log.Debugf("Overriding default package manager with value from Viper: %s", viperDefault)
		switch viperDefault {
		case "choco":
			osInfo.PackageManager.Default = osInfo.PackageManager.Chocolatey
		case "scoop":
			osInfo.PackageManager.Default = osInfo.PackageManager.Scoop
		default:
			log.Warnf("Unknown default package manager specified in Viper config: %s", viperDefault)
		}
	}
}
