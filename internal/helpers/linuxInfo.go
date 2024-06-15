package helpers

import (
	"fmt"
	"github.com/fynxlabs/rwr/internal/types"
	"os"
	"strings"

	"github.com/charmbracelet/log"
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

		log.Debugf("Found package manager for distro %s: %s", distro, pm)
		return pm
	}
	log.Debugf("No package manager found for distro %s", distro)
	return ""
}

func SetLinuxDetails(osInfo *types.OSInfo) error {
	log.Debug("Setting Linux package manager details.")

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

	defaultPackageManager := getDefaultPackageManagerFromOSRelease()
	if defaultPackageManager != "" {
		log.Debugf("Default package manager from OS release: %s", defaultPackageManager)
		setPackageManagerDetails(osInfo, defaultPackageManager)
	}

	return nil
}

func setPackageManagerDetails(osInfo *types.OSInfo, pm string) {
	switch pm {
	case "apt":
		binPath, err := GetBinPath("apt")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding apt binary path: %v", err)
			return
		}

		// Check if nala is installed and use it instead of apt
		if CommandExists("nala") {
			nalaBinPath, err := GetBinPath("nala")
			log.Debugf("%s bin path: %s", pm, nalaBinPath)
			if err != nil {
				log.Warnf("Error finding nala binary path: %v", err)
				return
			}

			osInfo.PackageManager.Apt = types.PackageManagerInfo{
				Name:     "apt",
				Bin:      nalaBinPath,
				List:     fmt.Sprintf("%s list --installed", nalaBinPath),
				Search:   fmt.Sprintf("%s search", nalaBinPath),
				Install:  fmt.Sprintf("%s install -y", nalaBinPath),
				Remove:   fmt.Sprintf("%s remove -y", nalaBinPath),
				Update:   fmt.Sprintf("%s update && %s upgrade -y", nalaBinPath, nalaBinPath),
				Clean:    fmt.Sprintf("%s clean", nalaBinPath),
				Elevated: true,
			}
			log.Debugf("Setting nala package manager")
			osInfo.PackageManager.Default = osInfo.PackageManager.Apt
			log.Debugf("Using nala package manager for apt, also setting as default")
		} else {
			osInfo.PackageManager.Apt = types.PackageManagerInfo{
				Name:     "apt",
				Bin:      binPath,
				List:     "dpkg --get-selections",
				Search:   fmt.Sprintf("%s search", binPath),
				Install:  fmt.Sprintf("%s install -y", binPath),
				Remove:   fmt.Sprintf("%s remove -y", binPath),
				Update:   fmt.Sprintf("%s update && %s upgrade -y", binPath, binPath),
				Clean:    fmt.Sprintf("%s clean", binPath),
				Elevated: true,
			}
			log.Debugf("Setting apt package manager")
			osInfo.PackageManager.Default = osInfo.PackageManager.Apt
			log.Debugf("Using apt package manager as default")
		}
	case "dnf":
		binPath, err := GetBinPath("dnf")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding dnf binary path: %v", err)
			return
		}

		osInfo.PackageManager.Dnf = types.PackageManagerInfo{
			Name:     "dnf",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list installed", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -y", binPath),
			Remove:   fmt.Sprintf("%s remove -y", binPath),
			Update:   fmt.Sprintf("%s update -y", binPath),
			Clean:    fmt.Sprintf("%s clean all", binPath),
			Elevated: true,
		}
		log.Debugf("Setting dnf package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Dnf
		log.Debugf("Using dnf package manager as default")
	case "yum":
		binPath, err := GetBinPath("yum")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding yum binary path: %v", err)
			return
		}

		osInfo.PackageManager.Yum = types.PackageManagerInfo{
			Name:     "yum",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list installed", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -y", binPath),
			Remove:   fmt.Sprintf("%s remove -y", binPath),
			Update:   fmt.Sprintf("%s update -y", binPath),
			Clean:    fmt.Sprintf("%s clean all", binPath),
			Elevated: true,
		}
	case "eopkg":
		binPath, err := GetBinPath("eopkg")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding eopkg binary path: %v", err)
			return
		}

		osInfo.PackageManager.Eopkg = types.PackageManagerInfo{
			Name:     "eopkg",
			Bin:      binPath,
			List:     fmt.Sprintf("%s li", binPath),
			Search:   fmt.Sprintf("%s sr", binPath),
			Install:  fmt.Sprintf("%s it -y", binPath),
			Remove:   fmt.Sprintf("%s rm -y", binPath),
			Update:   fmt.Sprintf("%s ur", binPath),
			Clean:    fmt.Sprintf("%s rmo -y", binPath),
			Elevated: true,
		}
		log.Debugf("Setting eopkg package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Eopkg
		log.Debugf("Using eopkg package manager as default")
	case "pacman":
		binPath, err := GetBinPath("pacman")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding pacman binary path: %v", err)
			return
		}

		osInfo.PackageManager.Pacman = types.PackageManagerInfo{
			Name:     "pacman",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -Sy --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: true,
		}
		log.Debugf("Setting pacman package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Pacman
		log.Debugf("Using pacman package manager as default")
	case "yay":
		binPath, err := GetBinPath("yay")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding yay binary path: %v", err)
			return
		}

		osInfo.PackageManager.Yay = types.PackageManagerInfo{
			Name:     "yay",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -S --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: false,
		}
		log.Debugf("Setting yay package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Yay
		log.Debugf("Using yay package manager as default")
	case "paru":
		binPath, err := GetBinPath("paru")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding paru binary path: %v", err)
			return
		}

		osInfo.PackageManager.Paru = types.PackageManagerInfo{
			Name:     "paru",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -S --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: false,
		}
		log.Debugf("Setting paru package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Paru
		log.Debugf("Using paru package manager as default")
	case "trizen":
		binPath, err := GetBinPath("trizen")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding trizen binary path: %v", err)
			return
		}

		osInfo.PackageManager.Trizen = types.PackageManagerInfo{
			Name:     "trizen",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -S --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: false,
		}
		log.Debugf("Setting trizen package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Trizen
		log.Debugf("Using trizen package manager as default")
	case "yaourt":
		binPath, err := GetBinPath("yaourt")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding yaourt binary path: %v", err)
			return
		}

		osInfo.PackageManager.Yaourt = types.PackageManagerInfo{
			Name:     "yaourt",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -S --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: false,
		}
		log.Debugf("Setting yaourt package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Yaourt
		log.Debugf("Using yaourt package manager as default")
	case "pamac":
		binPath, err := GetBinPath("pamac")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding pamac binary path: %v", err)
			return
		}

		osInfo.PackageManager.Pamac = types.PackageManagerInfo{
			Name:     "pamac",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list -i", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -y", binPath),
			Remove:   fmt.Sprintf("%s remove -y", binPath),
			Update:   fmt.Sprintf("%s update", binPath),
			Clean:    fmt.Sprintf("%s clean -y", binPath),
			Elevated: false,
		}
		log.Debugf("Setting pamac package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Pamac
		log.Debugf("Using pamac package manager as default")
	case "aura":
		binPath, err := GetBinPath("aura")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding aura binary path: %v", err)
			return
		}

		osInfo.PackageManager.Aura = types.PackageManagerInfo{
			Name:     "aura",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -Q", binPath),
			Search:   fmt.Sprintf("%s -Ss", binPath),
			Install:  fmt.Sprintf("%s -A --noconfirm", binPath),
			Remove:   fmt.Sprintf("%s -R --noconfirm", binPath),
			Update:   fmt.Sprintf("%s -Syu --noconfirm", binPath),
			Clean:    fmt.Sprintf("%s -Sc --noconfirm", binPath),
			Elevated: false,
		}
		log.Debugf("Setting aura package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Aura
		log.Debugf("Using aura package manager as default")
	case "zypper":
		binPath, err := GetBinPath("zypper")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding zypper binary path: %v", err)
			return
		}

		osInfo.PackageManager.Zypper = types.PackageManagerInfo{
			Name:     "zypper",
			Bin:      binPath,
			List:     fmt.Sprintf("%s packages --installed-only", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -y", binPath),
			Remove:   fmt.Sprintf("%s remove -y", binPath),
			Update:   fmt.Sprintf("%s update -y", binPath),
			Clean:    fmt.Sprintf("%s clean", binPath),
			Elevated: true,
		}
		log.Debugf("Setting zypper package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Zypper
		log.Debugf("Using zypper package manager as default")
	case "emerge":
		binPath, err := GetBinPath("emerge")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding emerge binary path: %v", err)
			return
		}

		osInfo.PackageManager.Emerge = types.PackageManagerInfo{
			Name:     "emerge",
			Bin:      binPath,
			List:     "qlist -I",
			Search:   fmt.Sprintf("%s -s", binPath),
			Install:  fmt.Sprintf("%s -qv", binPath),
			Remove:   fmt.Sprintf("%s -C", binPath),
			Update:   fmt.Sprintf("%s -uDN @world", binPath),
			Clean:    fmt.Sprintf("%s --depclean", binPath),
			Elevated: true,
		}
		log.Debugf("Setting emerge package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Emerge
		log.Debugf("Using emerge package manager as default")
	case "nix":
		binPath, err := GetBinPath("nix-env")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding nix-env binary path: %v", err)
			return
		}

		osInfo.PackageManager.Nix = types.PackageManagerInfo{
			Name:     "nix",
			Bin:      binPath,
			List:     fmt.Sprintf("%s -q", binPath),
			Search:   "nix search",
			Install:  fmt.Sprintf("%s -i", binPath),
			Remove:   fmt.Sprintf("%s -e", binPath),
			Update:   fmt.Sprintf("nix-channel --update && %s -u '*'", binPath),
			Clean:    "nix-collect-garbage -d",
			Elevated: false,
		}
		log.Debugf("Setting nix package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Nix
		log.Debugf("Using nix package manager as default")
	case "brew":
		binPath, err := GetBinPath("brew")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding brew binary path: %v", err)
			return
		}

		osInfo.PackageManager.Brew = types.PackageManagerInfo{
			Name:     "brew",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -fq", binPath),
			Remove:   fmt.Sprintf("%s uninstall -fq", binPath),
			Update:   fmt.Sprintf("%s update && %s upgrade", binPath, binPath),
			Clean:    fmt.Sprintf("%s cleanup -q", binPath),
			Elevated: false,
		}
		log.Debugf("Setting brew package manager")
	case "cargo":
		binPath, err := GetBinPath("cargo")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding cargo binary path: %v", err)
			return
		}

		osInfo.PackageManager.Cargo = types.PackageManagerInfo{
			Name:     "cargo",
			Bin:      binPath,
			List:     fmt.Sprintf("%s install --list", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install", binPath),
			Remove:   fmt.Sprintf("%s uninstall", binPath),
			Update:   fmt.Sprintf("%s install --force", binPath),
			Clean:    fmt.Sprintf("%s cache --autoclean", binPath),
			Elevated: false,
		}
		log.Debugf("Setting cargo package manager")
	case "snap":
		binPath, err := GetBinPath("snap")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding snap binary path: %v", err)
			return
		}

		osInfo.PackageManager.Snap = types.PackageManagerInfo{
			Name:     "snap",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list", binPath),
			Search:   fmt.Sprintf("%s find", binPath),
			Install:  fmt.Sprintf("%s install", binPath),
			Remove:   fmt.Sprintf("%s remove", binPath),
			Update:   fmt.Sprintf("%s refresh", binPath),
			Clean:    fmt.Sprintf("%s refresh", binPath),
			Elevated: false,
		}
		log.Debugf("Setting snap package manager")
	case "flatpak":
		binPath, err := GetBinPath("flatpak")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding flatpak binary path: %v", err)
			return
		}

		osInfo.PackageManager.Flatpak = types.PackageManagerInfo{
			Name:     "flatpak",
			Bin:      binPath,
			List:     fmt.Sprintf("%s list", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s install -y", binPath),
			Remove:   fmt.Sprintf("%s uninstall -y", binPath),
			Update:   fmt.Sprintf("%s update -y", binPath),
			Clean:    fmt.Sprintf("%s uninstall --unused -y", binPath),
			Elevated: false,
		}
		log.Debugf("Setting flatpak package manager")
	case "apk":
		binPath, err := GetBinPath("apk")
		log.Debugf("%s bin path: %s", pm, binPath)
		if err != nil {
			log.Warnf("Error finding apk binary path: %v", err)
			return
		}

		osInfo.PackageManager.Apk = types.PackageManagerInfo{
			Name:     "apk",
			Bin:      binPath,
			List:     fmt.Sprintf("%s info", binPath),
			Search:   fmt.Sprintf("%s search", binPath),
			Install:  fmt.Sprintf("%s add", binPath),
			Remove:   fmt.Sprintf("%s del", binPath),
			Update:   fmt.Sprintf("%s update && %s upgrade", binPath, binPath),
			Clean:    fmt.Sprintf("%s cache clean", binPath),
			Elevated: true,
		}
		log.Debugf("Setting apk package manager")
		osInfo.PackageManager.Default = osInfo.PackageManager.Apk
		log.Debugf("Using apk package manager as default")
	default:
		log.Warnf("Unknown package manager: %s", pm)
	}
}
