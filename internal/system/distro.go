package system

import (
	"os"
	"strings"

	"github.com/charmbracelet/log"
)

// Known distribution families and their variants
var distroFamilies = map[string][]string{
	"arch":      {"endeavouros", "manjaro", "artix", "garuda", "blackarch", "archbang", "archcraft", "arcolinux"},
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

// GetDistroFamily returns the base distribution family for a given distribution
// For example, "endeavouros" would return "arch"
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

	// Check ID_LIKE in /etc/os-release for hints
	idLike := getDistroIDLike()
	if idLike != "" {
		for _, likeDistro := range strings.Split(idLike, " ") {
			if _, exists := distroFamilies[likeDistro]; exists {
				return likeDistro
			}
		}
	}

	// If no match found, return the original distro
	return distro
}

// IsDistroInFamily checks if a distribution is in a specific family
// For example, IsDistroInFamily("endeavouros", "arch") would return true
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

	// Check ID_LIKE in /etc/os-release
	idLike := getDistroIDLike()
	if idLike != "" && strings.Contains(idLike, family) {
		return true
	}

	return false
}

// getDistroIDLike returns the ID_LIKE field from /etc/os-release
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
