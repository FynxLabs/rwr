package helpers

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/thefynx/rwr/internal/types"
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
	case "yay":
		return osInfo.PackageManager.Yay, nil
	case "paru":
		return osInfo.PackageManager.Paru, nil
	case "trizen":
		return osInfo.PackageManager.Trizen, nil
	case "yaourt":
		return osInfo.PackageManager.Yaourt, nil
	case "pamac":
		return osInfo.PackageManager.Pamac, nil
	case "yum":
		return osInfo.PackageManager.Yum, nil
	case "aura":
		return osInfo.PackageManager.Aura, nil
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

func InstallOpenSSL(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var installCmd types.Command

	switch osInfo.OS {
	case "linux":
		log.Debugf("Installing OpenSSL on %s", osInfo.OS)
		switch osInfo.PackageManager.Default.Name {
		case "apt":
			log.Debugf("Installing OpenSSL with %s", osInfo.PackageManager.Apt.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Apt.Install,
				Args:     []string{"openssl", "libssl-dev"},
				Elevated: osInfo.PackageManager.Apt.Elevated,
			}
		case "dnf":
			log.Debugf("Installing OpenSSL with %s", osInfo.PackageManager.Dnf.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Dnf.Install,
				Args:     append([]string{"openssl", "openssl-devel"}),
				Elevated: osInfo.PackageManager.Dnf.Elevated,
			}
		case "yum":
			log.Debugf("Installing OpenSSL with %s", osInfo.PackageManager.Yum.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Yum.Install,
				Args:     append([]string{"openssl", "openssl-devel"}),
				Elevated: osInfo.PackageManager.Yum.Elevated,
			}
		case "pacman":
			log.Debugf("Installing OpenSSL with %s", osInfo.PackageManager.Pacman.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Pacman.Install,
				Args:     append([]string{"openssl"}),
				Elevated: osInfo.PackageManager.Pacman.Elevated,
			}
		case "zypper":
			log.Debugf("Installing OpenSSL with %s", osInfo.PackageManager.Zypper.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Zypper.Install,
				Args:     append([]string{"openssl", "libopenssl-devel"}),
				Elevated: osInfo.PackageManager.Zypper.Elevated,
			}
		default:
			log.Warnf("Unsupported package manager for OpenSSL installation: %s - Please manually install openssl", osInfo.PackageManager.Default.Name)
			return nil
		}
	case "macos":
		log.Debugf("Installing OpenSSL on %s", osInfo.OS)
		if osInfo.PackageManager.Default.Name == "brew" {
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Brew.Install,
				Args:     []string{"openssl"},
				Elevated: osInfo.PackageManager.Brew.Elevated,
			}
		} else {
			log.Warnf("Unsupported package manager for OpenSSL installation: %s - Please manually install openssl", osInfo.PackageManager.Default.Name)
			return nil
		}
	case "windows":
		log.Debugf("Installing OpenSSL on %s", osInfo.OS)
		if osInfo.PackageManager.Default.Name == "chocolatey" {
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Chocolatey.Install,
				Args:     []string{"openssl"},
				Elevated: osInfo.PackageManager.Chocolatey.Elevated,
			}
		} else {
			log.Warnf("Unsupported package manager for OpenSSL installation: %s - Please manually install openssl", osInfo.PackageManager.Default.Name)
			return nil
		}
	default:
		log.Warnf("Unsupported OS for OpenSSL installation: %s", osInfo.OS)
		return nil
	}

	err := RunCommand(installCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error installing OpenSSL: %v", err)
	}

	return nil
}

// InstallBuildEssentials installs build essentials on the system.
func InstallBuildEssentials(osInfo *types.OSInfo, initConfig *types.InitConfig) error {
	var installCmd types.Command

	switch osInfo.OS {
	case "linux":
		log.Debugf("Installing build essentials on %s", osInfo.OS)
		switch osInfo.PackageManager.Default.Name {
		case "apt":
			log.Debugf("Installing build essentials with %s", osInfo.PackageManager.Apt.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Apt.Install,
				Args:     []string{"build-essential", "cmake", "pkg-config", "libfreetype6-dev", "libfontconfig1-dev", "libxcb-xfixes0-dev", "libxkbcommon-dev", "python3"},
				Elevated: osInfo.PackageManager.Apt.Elevated,
			}
		case "dnf":
			log.Debugf("Installing build essentials with %s", osInfo.PackageManager.Dnf.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Dnf.Install,
				Args:     append([]string{"make", "cmake", "freetype-devel", "fontconfig-devel", "libxcb-devel", "libxkbcommon-devel", "g++"}),
				Elevated: osInfo.PackageManager.Dnf.Elevated,
			}
		case "yum":
			log.Debugf("Installing build essentials with %s", osInfo.PackageManager.Yum.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Yum.Install,
				Args:     append([]string{"make", "cmake", "freetype-devel", "fontconfig-devel", "libxcb-devel", "libxkbcommon-devel", "xcb-util-devel"}),
				Elevated: osInfo.PackageManager.Yum.Elevated,
			}
		case "pacman":
			log.Debugf("Installing build essentials with %s", osInfo.PackageManager.Pacman.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Pacman.Install,
				Args:     append([]string{"base-devel", "cmake", "freetype2", "fontconfig", "pkg-config", "libxcb", "libxkbcommon", "python"}),
				Elevated: osInfo.PackageManager.Pacman.Elevated,
			}
		case "zypper":
			log.Debugf("Installing build essentials with %s", osInfo.PackageManager.Zypper.Bin)
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Zypper.Install,
				Args:     append([]string{"make", "cmake", "freetype-devel", "fontconfig-devel", "libxcb-devel", "libxkbcommon-devel"}),
				Elevated: osInfo.PackageManager.Zypper.Elevated,
			}
		default:
			log.Warnf("Unsupported package manager for build essentials installation: %s - Please manually install build essentials", osInfo.PackageManager.Default.Name)
			return nil
		}
	case "macos":
		log.Debugf("Installing build essentials on %s", osInfo.OS)
		if osInfo.PackageManager.Default.Name == "brew" {
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Brew.Install,
				Args:     []string{"make", "cmake", "pkg-config", "freetype", "fontconfig"},
				Elevated: osInfo.PackageManager.Brew.Elevated,
			}
		} else {
			log.Warnf("Unsupported package manager for build essentials installation: %s - Please manually install build essentials", osInfo.PackageManager.Default.Name)
			return nil
		}
	case "windows":
		log.Debugf("Installing build essentials on %s", osInfo.OS)
		if osInfo.PackageManager.Default.Name == "chocolatey" {
			installCmd = types.Command{
				Exec:     osInfo.PackageManager.Chocolatey.Install,
				Args:     []string{"make", "cmake", "freetype", "fontconfig"},
				Elevated: osInfo.PackageManager.Chocolatey.Elevated,
			}
		} else {
			log.Warnf("Unsupported package manager for build essentials installation: %s - Please manually install build essentials", osInfo.PackageManager.Default.Name)
			return nil
		}
	default:
		log.Warnf("Unsupported OS for build essentials installation: %s", osInfo.OS)
		return nil
	}

	err := RunCommand(installCmd, initConfig.Variables.Flags.Debug)
	if err != nil {
		return fmt.Errorf("error installing build essentials: %v", err)
	}

	return nil
}
