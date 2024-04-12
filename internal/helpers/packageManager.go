package helpers

import (
	"fmt"
	"github.com/thefynx/rwr/internal/processors/types"
	"reflect"
	"strings"
)

func GetPackageManagerInfo(osInfo *types.OSInfo, pm string) (types.PackageManagerInfo, error) {
	switch pm {
	case "apt":
		return osInfo.PackageManager.Apt, nil
	case "dnf":
		return osInfo.PackageManager.Dnf, nil
	case "eopkg":
		return osInfo.PackageManager.Eopkg, nil
	case "yay", "paru", "trizen", "yaourt", "pamac", "aura":
		return osInfo.PackageManager.Default, nil
	case "pacman":
		return osInfo.PackageManager.Pacman, nil
	case "zypper":
		return osInfo.PackageManager.Zypper, nil
	case "emerge":
		return osInfo.PackageManager.Emerge, nil
	case "nix":
		return osInfo.PackageManager.Nix, nil
	case "brew":
		return osInfo.PackageManager.Brew, nil
	case "cargo":
		return osInfo.PackageManager.Cargo, nil
	case "snap":
		return osInfo.PackageManager.Snap, nil
	case "flatpak":
		return osInfo.PackageManager.Flatpak, nil
	case "apk":
		return osInfo.PackageManager.Apk, nil
	case "chocolatey":
		return osInfo.PackageManager.Chocolatey, nil
	case "scoop":
		return osInfo.PackageManager.Scoop, nil
	case "mas":
		return osInfo.PackageManager.MAS, nil
	default:
		return types.PackageManagerInfo{}, fmt.Errorf("unsupported package manager: %s", pm)
	}
}

func getPackageManagerNames(pm types.PackageManager) []string {
	var packageManagers []string
	val := reflect.ValueOf(pm)

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		if field.Type == reflect.TypeOf(types.PackageManagerInfo{}) {
			packageManagers = append(packageManagers, strings.ToLower(field.Name))
		}
	}

	return packageManagers
}
