package helpers

import (
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

func setLinuxDetails(osInfo *OSInfo) {
	log.Debug("Setting Linux package manager details.")

	defaultPackageManager := getDefaultPackageManagerFromOSRelease()
	if defaultPackageManager != "" {
		log.Debugf("Default package manager from OS release: %s", defaultPackageManager)
		setPackageManagerDetails(osInfo, defaultPackageManager)
	}

	for _, pm := range []string{"apt", "dnf", "eopkg", "yay", "paru", "trizen", "yaourt", "pamac", "aura", "pacman", "zypper", "emerge", "nix", "brew"} {
		if commandExists(pm) {
			setPackageManagerDetails(osInfo, pm)
		}
	}

	// Override default package manager if set in viper config
	viperDefault := viper.GetString("packageManager.linux.default")
	if viperDefault != "" {
		log.Debugf("Overriding default package manager with value from Viper: %s", viperDefault)
		setPackageManagerDetails(osInfo, viperDefault)
	}
}

func setPackageManagerDetails(osInfo *OSInfo, pm string) {
	switch pm {
	case "apt":
		osInfo.PackageManager.Apt = PackageManagerInfo{
			Bin:      "apt",
			List:     "dpkg --get-selections",
			Search:   "apt search",
			Install:  "apt install -y",
			Remove:   "apt remove -y",
			Clean:    "apt clean",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Apt
	case "dnf":
		osInfo.PackageManager.Dnf = PackageManagerInfo{
			Bin:      "dnf",
			List:     "dnf list installed",
			Search:   "dnf search",
			Install:  "dnf install -y",
			Remove:   "dnf remove -y",
			Clean:    "dnf clean all",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Dnf
	case "eopkg":
		osInfo.PackageManager.Eopkg = PackageManagerInfo{
			Bin:      "eopkg",
			List:     "eopkg li",
			Search:   "eopkg sr",
			Install:  "eopkg it -y",
			Remove:   "eopkg rm -y",
			Clean:    "eopkg rmo -y",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Eopkg
	case "yay":
		osInfo.PackageManager.Yay = PackageManagerInfo{
			Bin:      "yay",
			List:     "yay -Q",
			Search:   "yay -Ss",
			Install:  "yay -S --noconfirm",
			Remove:   "yay -R --noconfirm",
			Clean:    "yay -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Yay
	case "paru":
		osInfo.PackageManager.Paru = PackageManagerInfo{
			Bin:      "paru",
			List:     "paru -Q",
			Search:   "paru -Ss",
			Install:  "paru -S --noconfirm",
			Remove:   "paru -R --noconfirm",
			Clean:    "paru -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Paru
	case "trizen":
		osInfo.PackageManager.Trizen = PackageManagerInfo{
			Bin:      "trizen",
			List:     "trizen -Q",
			Search:   "trizen -Ss",
			Install:  "trizen -S --noconfirm",
			Remove:   "trizen -R --noconfirm",
			Clean:    "trizen -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Trizen
	case "yaourt":
		osInfo.PackageManager.Yaourt = PackageManagerInfo{
			Bin:      "yaourt",
			List:     "yaourt -Q",
			Search:   "yaourt -Ss",
			Install:  "yaourt -S --noconfirm",
			Remove:   "yaourt -R --noconfirm",
			Clean:    "yaourt -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Yaourt
	case "pamac":
		osInfo.PackageManager.Pamac = PackageManagerInfo{
			Bin:      "pamac",
			List:     "pamac list -i",
			Search:   "pamac search",
			Install:  "pamac install -y",
			Remove:   "pamac remove -y",
			Clean:    "pamac clean -y",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Pamac
	case "aura":
		osInfo.PackageManager.Aura = PackageManagerInfo{
			Bin:      "aura",
			List:     "aura -Q",
			Search:   "aura -Ss",
			Install:  "aura -A --noconfirm",
			Remove:   "aura -R --noconfirm",
			Clean:    "aura -Sc --noconfirm",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Aura
	case "pacman":
		osInfo.PackageManager.Pacman = PackageManagerInfo{
			Bin:      "pacman",
			List:     "pacman -Q",
			Search:   "pacman -Ss",
			Install:  "pacman -Sy --noconfirm",
			Remove:   "pacman -R --noconfirm",
			Clean:    "pacman -Sc --noconfirm",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Pacman
	case "zypper":
		osInfo.PackageManager.Zypper = PackageManagerInfo{
			Bin:      "zypper",
			List:     "zypper packages --installed-only",
			Search:   "zypper search",
			Install:  "zypper install -y",
			Remove:   "zypper remove -y",
			Clean:    "zypper clean",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Zypper
	case "emerge":
		osInfo.PackageManager.Emerge = PackageManagerInfo{
			Bin:      "emerge",
			List:     "qlist -I",
			Search:   "emerge -s",
			Install:  "emerge -qv",
			Remove:   "emerge -C",
			Clean:    "emerge --depclean",
			Elevated: true,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Emerge
	case "nix":
		osInfo.PackageManager.Nix = PackageManagerInfo{
			Bin:      "nix-env",
			List:     "nix-env -q",
			Search:   "nix search",
			Install:  "nix-env -i",
			Remove:   "nix-env -e",
			Clean:    "nix-collect-garbage -d",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Nix
	case "brew":
		osInfo.PackageManager.Brew = PackageManagerInfo{
			Bin:      "brew",
			List:     "brew list",
			Search:   "brew search",
			Install:  "brew install -fq",
			Remove:   "brew uninstall -fq",
			Clean:    "brew cleanup -q",
			Elevated: false,
		}
		osInfo.PackageManager.Default = osInfo.PackageManager.Brew
	default:
		log.Warnf("Unknown package manager: %s", pm)
	}
}
