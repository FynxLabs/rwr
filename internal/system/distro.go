package system

import (
	"os"
	"strings"

	"charm.land/log/v2"
)

// nativeManagers lists the package managers that belong to a distribution family,
// in the order they should be preferred as the default.
//
// This is the mapping that makes "which package manager does this machine use?"
// answerable without enumerating every derivative distribution. Families are few
// and stable; derivatives are many and keep appearing.
var nativeManagers = map[string][]string{
	"arch":      {"paru", "yay", "trizen", "aura", "pamac", "pacman"},
	"debian":    {"apt"},
	"ubuntu":    {"apt"},
	"fedora":    {"dnf"},
	"rhel":      {"dnf"},
	"suse":      {"zypper"},
	"alpine":    {"apk"},
	"void":      {"xbps"},
	"gentoo":    {"emerge"},
	"slackware": {"slackpkg"},
	"solus":     {"eopkg"},
}

// managerFamily maps a package manager back to the family it identifies.
//
// Used to infer the family from what is installed when /etc/os-release names a
// distribution nobody has heard of. A great many derivatives ship neither a known
// ID nor an ID_LIKE - PrismLinux reports ID=prismlinux and nothing else - but the
// presence of pacman and its database says "arch" unambiguously.
var managerFamily = map[string]string{
	"pacman":   "arch",
	"apt":      "debian",
	"dnf":      "fedora",
	"zypper":   "suse",
	"apk":      "alpine",
	"xbps":     "void",
	"emerge":   "gentoo",
	"slackpkg": "slackware",
	"eopkg":    "solus",
}

// Known distribution families and their variants.
var distroFamilies = map[string][]string{
	"arch":      {"endeavouros", "manjaro", "artix", "garuda", "blackarch", "archbang", "archcraft", "arcolinux", "acreetion"},
	"debian":    {"ubuntu", "elementary", "zorin", "kali", "parrot", "mx", "deepin", "devuan"},
	"ubuntu":    {"kubuntu", "xubuntu", "lubuntu", "pop-os", "ubuntu-mate", "linuxmint", "ubuntu-budgie", "ubuntu-studio", "edubuntu", "mythbuntu"},
	"fedora":    {"nobara"},
	"rhel":      {"almalinux", "rocky", "oracle"},
	"suse":      {"opensuse", "opensuse-leap", "opensuse-tumbleweed"},
	"gentoo":    {"funtoo", "chromeos"},
	"slackware": {"slax", "zenwalk", "vector"},
	"void":      {"void-live"},
	"alpine":    {"postmarketos"},
}

// GetDistroFamily returns the base distribution family for a given distribution.
// For example, "endeavouros" returns "arch", and "linuxmint" returns "ubuntu".
// Falls back to checking ID_LIKE in /etc/os-release if no direct match is found.
func GetDistroFamily(distro string) string {
	// If the distro is a base distro itself, return it
	if _, exists := distroFamilies[distro]; exists {
		return distro
	}

	// Check if the distro is a variant of a known base distro
	for baseDistro, variants := range distroFamilies {
		for _, variant := range variants {
			if distro == variant {
				return baseDistro
			}
		}
	}

	// Fall back to this machine's ID_LIKE, but only when the distro being asked
	// about is this machine. See idLikeFor.
	if idLike := idLikeFor(distro); idLike != "" {
		for _, likeDistro := range strings.Split(idLike, " ") {
			if _, exists := distroFamilies[likeDistro]; exists {
				return likeDistro
			}
		}
	}

	// If no match found, return the original distro
	return distro
}

// IsDistroInFamily reports whether a distribution belongs to a specific family.
// For example, IsDistroInFamily("endeavouros", "arch") returns true.
func IsDistroInFamily(distro, family string) bool {
	// If the distro is the family itself, return true
	if distro == family {
		return true
	}

	// Check if the distro is a variant of the family
	variants, exists := distroFamilies[family]
	if exists {
		for _, variant := range variants {
			if distro == variant {
				return true
			}
		}
	}

	// Fall back to this machine's ID_LIKE, but only when the distro being asked
	// about is this machine. See idLikeFor.
	if idLike := idLikeFor(distro); idLike != "" && strings.Contains(idLike, family) {
		return true
	}

	return false
}

// idLikeFor returns this machine's ID_LIKE, but only when distro names this
// machine's own distribution.
//
// ID_LIKE describes the host and nothing else. Consulting it for an arbitrary
// distro answered every question in terms of whatever the host happens to be:
// on a Debian derivative (ID_LIKE=debian), IsDistroInFamily("arch", "debian")
// returned true and GetDistroFamily on any unknown distro returned "debian".
// That silently mapped unrecognised distributions onto the host's family.
var idLikeFor = func(distro string) string {
	if !strings.EqualFold(distro, getLinuxDistro()) {
		return ""
	}
	return getDistroIDLike()
}

// getDistroIDLike returns the ID_LIKE field from /etc/os-release.
func getDistroIDLike() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		log.Debugf("Error reading /etc/os-release: %v", err)
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID_LIKE=") {
			return strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}

	return ""
}
