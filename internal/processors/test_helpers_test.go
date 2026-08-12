package processors

import (
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
)

// containsString checks if a string contains a substring.
func containsString(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// givePackageManager records one detected package manager on a hand-built
// OSInfo.
//
// All() treats a darwin machine with no detected package manager as a machine
// that needs one installed, and goes off to install Homebrew before it reaches
// the processor loop. That is right for a real Mac and wrong for a fixture: a
// test about the run loop should not be describing a Mac with no package
// manager at all, because no such Mac ever reaches All() in practice - DetectOS
// fills this in.
//
// Leaving it empty is what made every All()-driven test fail on darwin with
// "no provider definition for package manager brew" while passing on Linux,
// where that branch does not exist.
func givePackageManager(osInfo *types.OSInfo, bin string) {
	info := types.PackageManagerInfo{Name: "pacman", Bin: bin}
	osInfo.PackageManager.Default = info
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{"pacman": info}
}
