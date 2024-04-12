package helpers

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/thefynx/rwr/internal/processors/types"
)

// SetWindowsDetails Sets the package manager details for Windows.
func SetWindowsDetails(osInfo *types.OSInfo) {
	log.Debug("Setting Windows package manager details.")

	//TODO: Move all package manager actions to a separate file to avoid duplication
	if CommandExists("choco") {
		log.Debug("Chocolatey detected.")
		osInfo.PackageManager.Chocolatey = types.PackageManagerInfo{
			Bin:     "choco",
			List:    "choco list --local-only",
			Search:  "choco search",
			Install: "choco install -y",
			Remove:  "choco uninstall -y",
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
			Remove:  "scoop uninstall",
			Update:  "scoop update",
			Clean:   "scoop cache rm *",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Scoop
		}
	}

	if CommandExists("winget") {
		log.Debug("Winget detected.")
		osInfo.PackageManager.Winget = types.PackageManagerInfo{
			Bin:     "winget",
			List:    "winget list",
			Search:  "winget search",
			Install: "winget install",
			Remove:  "winget uninstall",
			Update:  "winget upgrade",
			Clean:   "winget clean",
		}
		if osInfo.PackageManager.Default.Bin == "" {
			osInfo.PackageManager.Default = osInfo.PackageManager.Winget
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
		case "winget":
			osInfo.PackageManager.Default = osInfo.PackageManager.Winget
		default:
			log.Warnf("Unknown default package manager specified in Viper config: %s", viperDefault)
		}
	}
}
