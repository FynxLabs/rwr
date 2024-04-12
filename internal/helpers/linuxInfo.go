package helpers

import (
	"github.com/thefynx/rwr/internal/processors/types"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

var packageManagerMap = map[string]string{
	"arch":      "pacman",
	"debian":    "apt",
	"ubuntu":    "apt",
	"fedora":    "dnf",
	"rhel":      "yum",
	"centos":    "yum",
	"opensuse":  "zypper",
	"suse":      "zypper",
	"gentoo":    "emerge",
	"slackware": "slackpkg",
	"void":      "xbps",
	"solus":     "eopkg",
}

func getDefaultPackageManagerFromOSRelease() string {
	// Read the contents of the /etc/os-release file
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		log.Warnf("Error reading /etc/os-release file: %s", err)
		return ""
	}

	// Parse the contents of the file
	osRelease := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			osRelease[key] = value
		}
	}

	// Check the ID field first
	id := osRelease["ID"]
	if id != "" {
		if pm := getPackageManagerForDistro(id); pm != "" {
			return pm
		}
	}

	// If ID doesn't match any known distribution, check ID_LIKE
	idLike := osRelease["ID_LIKE"]
	if idLike != "" {
		for _, distro := range strings.Split(idLike, " ") {
			if pm := getPackageManagerForDistro(distro); pm != "" {
				return pm
			}
		}
	}

	// If no known distribution is found, return an empty string
	return ""
}

func getPackageManagerForDistro(distro string) string {
	if pm, ok := packageManagerMap[distro]; ok {
		return pm
	}
	return ""
}

func SetLinuxDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Linux package manager details.")

	defaultPackageManager := getDefaultPackageManagerFromOSRelease()
	if defaultPackageManager != "" {
		log.Debugf("Default package manager from OS release: %s", defaultPackageManager)
		setPackageManagerDetails(osInfo, defaultPackageManager)
	}

	packageManagers := getPackageManagerNames(osInfo.PackageManager)

	for _, pm := range packageManagers {
		_, err := GetPackageManagerInfo(osInfo, pm)
		if err != nil {
			log.Debugf("Package manager not found: %s", pm)
			continue
		}

		if CommandExists(pm) {
			log.Debugf("Package manager found: %s", pm)
			setPackageManagerDetails(osInfo, pm)
		} else {
			log.Debugf("Package manager not found: %s", pm)
		}
	}

	// Override default package manager if set in viper config
	viperDefault := viper.GetString("packageManager.linux.default")
	if viperDefault != "" {
		log.Debugf("Overriding default package manager with value from Viper: %s", viperDefault)
		setPackageManagerDetails(osInfo, viperDefault)
	}

	return nil
}

//TODO: Move all package manager actions to a separate file to avoid duplication

func setPackageManagerDetails(osInfo *types.OSInfo, pm string) {
	switch pm {
	case "apt":
		// Check if nala is installed and use it instead of apt
		if CommandExists("nala") {
			osInfo.PackageManager.Apt = types.PackageManagerInfo{
				Bin:      "nala",
				List:     "nala list --installed",
				Search:   "nala search",
				Install:  "nala install -y",
				Remove:   "nala remove -y",
				Update:   "nala update && nala upgrade -y",
				Clean:    "nala clean",
				Elevated: true,
			}
			osInfo.PackageManager.Default = osInfo.PackageManager.Apt
			log.Debugf("Using nala package manager for apt")
		} else {
			osInfo.PackageManager.Apt = types.PackageManagerInfo{
				Bin:      "apt",
				List:     "dpkg --get-selections",
				Search:   "apt search",
				Install:  "apt install -y",
				Remove:   "apt remove -y",
				Update:   "apt update && apt upgrade -y",
				Clean:    "apt clean",
				Elevated: true,
			}
			osInfo.PackageManager.Default = osInfo.PackageManager.Apt
			log.Debugf("Using apt package manager")
		}
	case "dnf":
		osInfo.PackageManager.Dnf = types.PackageManagerInfo{
			Bin:      "dnf",
			List:     "dnf list installed",
			Search:   "dnf search",
			Install:  "dnf install -y",
			Remove:   "dnf remove -y",
			Update:   "dnf update -y",
			Clean:    "dnf clean all",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Dnf
		log.Debugf("Using dnf package manager")
	case "eopkg":
		osInfo.PackageManager.Eopkg = types.PackageManagerInfo{
			Bin:      "eopkg",
			List:     "eopkg li",
			Search:   "eopkg sr",
			Install:  "eopkg it -y",
			Remove:   "eopkg rm -y",
			Update:   "eopkg ur",
			Clean:    "eopkg rmo -y",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Eopkg
		log.Debugf("Using eopkg package manager")
	case "pacman":
		osInfo.PackageManager.Pacman = types.PackageManagerInfo{
			Bin:      "pacman",
			List:     "pacman -Q",
			Search:   "pacman -Ss",
			Install:  "pacman -Sy --noconfirm",
			Remove:   "pacman -R --noconfirm",
			Update:   "pacman -Syu --noconfirm",
			Clean:    "pacman -Sc --noconfirm",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Pacman
		log.Debugf("Using pacman package manager")
	case "yay":
		osInfo.PackageManager.Yay = types.PackageManagerInfo{
			Bin:      "yay",
			List:     "yay -Q",
			Search:   "yay -Ss",
			Install:  "yay -S --noconfirm",
			Remove:   "yay -R --noconfirm",
			Update:   "yay -Syu --noconfirm",
			Clean:    "yay -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Yay
		log.Debugf("Using yay package manager")
	case "paru":
		osInfo.PackageManager.Paru = types.PackageManagerInfo{
			Bin:      "paru",
			List:     "paru -Q",
			Search:   "paru -Ss",
			Install:  "paru -S --noconfirm",
			Remove:   "paru -R --noconfirm",
			Update:   "paru -Syu --noconfirm",
			Clean:    "paru -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Paru
		log.Debugf("Using paru package manager")
	case "trizen":
		osInfo.PackageManager.Trizen = types.PackageManagerInfo{
			Bin:      "trizen",
			List:     "trizen -Q",
			Search:   "trizen -Ss",
			Install:  "trizen -S --noconfirm",
			Remove:   "trizen -R --noconfirm",
			Update:   "trizen -Syu --noconfirm",
			Clean:    "trizen -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Trizen
		log.Debugf("Using trizen package manager")
	case "yaourt":
		osInfo.PackageManager.Yaourt = types.PackageManagerInfo{
			Bin:      "yaourt",
			List:     "yaourt -Q",
			Search:   "yaourt -Ss",
			Install:  "yaourt -S --noconfirm",
			Remove:   "yaourt -R --noconfirm",
			Update:   "yaourt -Syu --noconfirm",
			Clean:    "yaourt -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Yaourt
		log.Debugf("Using yaourt package manager")
	case "pamac":
		osInfo.PackageManager.Pamac = types.PackageManagerInfo{
			Bin:      "pamac",
			List:     "pamac list -i",
			Search:   "pamac search",
			Install:  "pamac install -y",
			Remove:   "pamac remove -y",
			Update:   "pamac update",
			Clean:    "pamac clean -y",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Pamac
		log.Debugf("Using pamac package manager")
	case "aura":
		osInfo.PackageManager.Aura = types.PackageManagerInfo{
			Bin:      "aura",
			List:     "aura -Q",
			Search:   "aura -Ss",
			Install:  "aura -A --noconfirm",
			Remove:   "aura -R --noconfirm",
			Update:   "aura -Syu --noconfirm",
			Clean:    "aura -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Aura
		log.Debugf("Using aura package manager")
	case "zypper":
		osInfo.PackageManager.Zypper = types.PackageManagerInfo{
			Bin:      "zypper",
			List:     "zypper packages --installed-only",
			Search:   "zypper search",
			Install:  "zypper install -y",
			Remove:   "zypper remove -y",
			Update:   "zypper update -y",
			Clean:    "zypper clean",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Zypper
		log.Debugf("Using zypper package manager")
	case "emerge":
		osInfo.PackageManager.Emerge = types.PackageManagerInfo{
			Bin:      "emerge",
			List:     "qlist -I",
			Search:   "emerge -s",
			Install:  "emerge -qv",
			Remove:   "emerge -C",
			Update:   "emerge -uDN @world",
			Clean:    "emerge --depclean",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Emerge
		log.Debugf("Using emerge package manager")
	case "nix":
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
		osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		log.Debugf("Using nix package manager")
	case "brew":
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
	case "cargo":
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
	default:
		log.Warnf("Unknown package manager: %s", pm)
	}
}
