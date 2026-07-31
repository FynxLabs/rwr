package system

import (
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func pm(name string) types.PackageManagerInfo {
	return types.PackageManagerInfo{Name: name, Bin: "/usr/bin/" + name}
}

// The real embedded provider definitions must match the distributions they claim.
// This is the bug class that excluded apt/dnf/pacman on every Linux system: the
// per-OS setup filtered with a literal "linux" string match, so every provider
// naming concrete distros (apt lists debian/ubuntu, pacman lists arch/cachyos/...)
// was dropped. supportsSystem is the matcher that gets it right, and this test
// pins that the shipped definitions actually resolve for real distributions.
func TestSupportsSystem_MatchesShippedProvidersForRealDistros(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	tests := []struct {
		distro   string
		wantable []string
	}{
		{"debian", []string{"apt", "flatpak", "snap"}},
		{"ubuntu", []string{"apt", "flatpak", "snap"}},
		{"fedora", []string{"dnf", "flatpak"}},
		{"arch", []string{"pacman", "paru", "yay", "flatpak"}},
		{"cachyos", []string{"pacman", "paru", "yay"}},
		{"manjaro", []string{"pacman", "paru"}},
		{"opensuse-tumbleweed", []string{"zypper"}},
		{"alpine", []string{"apk"}},
	}

	for _, tt := range tests {
		t.Run(tt.distro, func(t *testing.T) {
			for _, name := range tt.wantable {
				provider, ok := loaded[name]
				if !ok {
					t.Fatalf("provider %q is not in the embedded definitions", name)
				}
				if !supportsSystem(provider, "linux", tt.distro) {
					t.Errorf("provider %q should support linux/%s (distributions=%v)",
						name, tt.distro, provider.Detection.Distributions)
				}
			}
		})
	}
}

// A provider must not be offered on a distro it does not support.
func TestSupportsSystem_RejectsWrongDistro(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	tests := []struct {
		provider string
		distro   string
	}{
		{"apt", "arch"},
		{"pacman", "debian"},
		{"dnf", "ubuntu"},
		{"apk", "fedora"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.distro, func(t *testing.T) {
			provider, ok := loaded[tt.provider]
			if !ok {
				t.Fatalf("provider %q is not in the embedded definitions", tt.provider)
			}
			if supportsSystem(provider, "linux", tt.distro) {
				t.Errorf("provider %q should not support linux/%s (distributions=%v)",
					tt.provider, tt.distro, provider.Detection.Distributions)
			}
		})
	}
}

// Providers that declare the "linux" wildcard stay available everywhere.
func TestSupportsSystem_WildcardLinuxProvidersMatchAnyDistro(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	for _, name := range []string{"flatpak", "nix", "cargo"} {
		provider, ok := loaded[name]
		if !ok {
			t.Fatalf("provider %q is not in the embedded definitions", name)
		}
		for _, distro := range []string{"debian", "arch", "fedora", "void"} {
			if !supportsSystem(provider, "linux", distro) {
				t.Errorf("wildcard provider %q should support linux/%s", name, distro)
			}
		}
	}
}

func TestSetDefaultManager_UsesPreferenceOrder(t *testing.T) {
	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"pacman": pm("pacman"),
		"yay":    pm("yay"),
		"paru":   pm("paru"),
	}

	setDefaultManager(osInfo, []string{"paru", "yay", "trizen", "aura", "pamac", "pacman"})

	if got := osInfo.PackageManager.Default.Name; got != "paru" {
		t.Errorf("default = %q, want paru (first present preference)", got)
	}
}

func TestSetDefaultManager_FallsThroughMissingPreferences(t *testing.T) {
	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"pacman": pm("pacman"),
	}

	setDefaultManager(osInfo, []string{"paru", "yay", "pacman"})

	if got := osInfo.PackageManager.Default.Name; got != "pacman" {
		t.Errorf("default = %q, want pacman", got)
	}
}

// Go randomizes map iteration, so an unsorted fallback could resolve a different
// default on every run — and with it, a different package manager for any package
// that did not name one.
func TestSetDefaultManager_FallbackIsDeterministic(t *testing.T) {
	managers := map[string]types.PackageManagerInfo{
		"snap":    pm("snap"),
		"flatpak": pm("flatpak"),
		"nix":     pm("nix"),
		"cargo":   pm("cargo"),
	}

	first := ""
	for i := 0; i < 100; i++ {
		osInfo := &types.OSInfo{}
		osInfo.PackageManager.Managers = managers

		setDefaultManager(osInfo, nil)

		got := osInfo.PackageManager.Default.Name
		if i == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("default package manager is not deterministic: got %q then %q", first, got)
		}
	}

	if first != "cargo" {
		t.Errorf("fallback default = %q, want cargo (alphabetically first)", first)
	}
}

func TestSetDefaultManager_NoManagersLeavesDefaultEmpty(t *testing.T) {
	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{}

	setDefaultManager(osInfo, []string{"brew"})

	if got := osInfo.PackageManager.Default.Name; got != "" {
		t.Errorf("default = %q, want empty", got)
	}
}

// mustLoadEmbedded returns the embedded provider definitions. Reading them
// directly means these assertions hold on any machine, whether or not the
// corresponding package manager binaries are installed.
func mustLoadEmbedded(t *testing.T) map[string]*types.Provider {
	t.Helper()
	loaded, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders: %v", err)
	}
	return loaded
}

// The regression this change fixes, stated directly: the per-OS setup used to
// keep only providers whose distribution list literally contained "linux". Every
// package manager that names concrete distributions was therefore dropped from
// osInfo.PackageManager.Managers, which is what CleanPackageManagers and the
// core-package (openssl, build-essentials) resolution read from.
//
// Fails against the old literal filter, passes against supportsSystem.
func TestProviderFilter_LiteralLinuxMatchWouldDropRealPackageManagers(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	// Package managers a Linux user actually installs things with.
	realManagers := []string{"apt", "dnf", "pacman", "zypper", "apk", "paru", "yay"}

	for _, name := range realManagers {
		provider, ok := loaded[name]
		if !ok {
			t.Fatalf("provider %q is not in the embedded definitions", name)
		}

		// This is the old predicate: a literal string match for "linux".
		keptByLiteralMatch := slices.Contains(provider.Detection.Distributions, "linux")
		if keptByLiteralMatch {
			t.Errorf("provider %q now declares the linux wildcard; this test no longer "+
				"covers the regression and should be revisited", name)
		}

		// This is the predicate in use, evaluated for a distro the provider serves.
		distro := provider.Detection.Distributions[0]
		if !supportsSystem(provider, "linux", distro) {
			t.Errorf("provider %q must be available on linux/%s", name, distro)
		}
	}
}

// End-to-end proof on a real machine: after OS detection, the distribution's own
// package manager must appear in osInfo.PackageManager.Managers. That map is what
// CleanPackageManagers and the core-package installers consume, and under the old
// literal "linux" filter it never contained apt/dnf/pacman/zypper/apk — so a
// system running `rwr all` reported "Cleaning up package managers" while never
// cleaning the one package manager it actually uses.
//
// Skips where none of these are installed, so it is meaningful in CI containers
// and on developer machines alike without being flaky.
func TestSetLinuxDetails_IncludesNativePackageManager(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}

	native := ""
	for _, name := range []string{"pacman", "apt", "dnf", "zypper", "apk", "xbps-install"} {
		if _, err := exec.LookPath(name); err == nil {
			native = name
			break
		}
	}
	if native == "" {
		t.Skip("no distribution package manager found on PATH")
	}

	// xbps-install is the binary; the provider is named xbps.
	providerName := native
	if native == "xbps-install" {
		providerName = "xbps"
	}

	osInfo := &types.OSInfo{}
	osInfo.System.OSFamily = getOSFamily()

	if err := SetLinuxDetails(osInfo); err != nil {
		t.Fatalf("SetLinuxDetails: %v", err)
	}

	if _, ok := osInfo.PackageManager.Managers[providerName]; !ok {
		t.Errorf("native package manager %q missing from Managers; got %v",
			providerName, managerNames(osInfo))
	}
}

func managerNames(osInfo *types.OSInfo) []string {
	names := make([]string, 0, len(osInfo.PackageManager.Managers))
	for name := range osInfo.PackageManager.Managers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// A provider that names an operating system must only surface on that OS.
func TestTargetsOS(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	tests := []struct {
		provider string
		os       string
		want     bool
	}{
		{"winget", "windows", true},
		{"winget", "linux", false},
		{"scoop", "darwin", false},
		{"brew", "darwin", true},
		{"brew", "linux", true},
		{"flatpak", "linux", true},
		{"flatpak", "windows", false},
		// Distro-scoped providers name no OS token, so the binary and file
		// evidence decides; those paths do not exist on the wrong OS.
		{"pacman", "linux", true},
		{"apt", "linux", true},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.os, func(t *testing.T) {
			provider, ok := loaded[tt.provider]
			if !ok {
				t.Fatalf("provider %q is not in the embedded definitions", tt.provider)
			}
			if got := targetsOS(provider, tt.os); got != tt.want {
				t.Errorf("targetsOS(%s, %s) = %v, want %v (distributions=%v)",
					tt.provider, tt.os, got, tt.want, provider.Detection.Distributions)
			}
		})
	}
}

// Every distro-scoped package manager must declare file evidence, because that is
// what makes it detectable on a derivative distribution nobody has listed. A
// provider without it can only ever be found by exact name match.
func TestEmbeddedProviders_DistroScopedProvidersDeclareFileEvidence(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	for _, name := range []string{"apt", "dnf", "pacman", "zypper", "apk", "paru", "yay", "xbps"} {
		provider, ok := loaded[name]
		if !ok {
			t.Fatalf("provider %q is not in the embedded definitions", name)
		}
		if len(provider.Detection.Files) == 0 {
			t.Errorf("provider %q declares no detection files, so it cannot be detected "+
				"on an unlisted derivative distribution", name)
		}
	}
}

// Family inference is what lets an unlisted derivative still get the right
// default. GetDistroFamily handles the distributions we know; anything else is
// resolved from the package manager that is actually installed.
func TestLinuxFamily(t *testing.T) {
	tests := []struct {
		name     string
		osFamily string
		managers []string
		want     string
	}{
		{
			name:     "known distro resolves from its name",
			osFamily: "debian",
			managers: []string{"apt", "flatpak"},
			want:     "debian",
		},
		{
			name:     "known derivative resolves through its family",
			osFamily: "endeavouros",
			managers: []string{"pacman"},
			want:     "arch",
		},
		{
			// PrismLinux: ID=prismlinux, no ID_LIKE, listed by nobody.
			name:     "unlisted derivative is inferred from the installed manager",
			osFamily: "prismlinux",
			managers: []string{"flatpak", "pacman", "yay"},
			want:     "arch",
		},
		{
			name:     "unlisted debian derivative is inferred from apt",
			osFamily: "somethingnew",
			managers: []string{"apt", "snap"},
			want:     "debian",
		},
		{
			name:     "no native manager means no family",
			osFamily: "mysterylinux",
			managers: []string{"flatpak", "nix"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osInfo := &types.OSInfo{}
			osInfo.System.OSFamily = tt.osFamily
			osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{}
			for _, m := range tt.managers {
				osInfo.PackageManager.Managers[m] = pm(m)
			}

			if got := linuxFamily(osInfo); got != tt.want {
				t.Errorf("linuxFamily(%q, managers=%v) = %q, want %q",
					tt.osFamily, tt.managers, got, tt.want)
			}
		})
	}
}

// An Arch derivative nobody has listed must still prefer its AUR helper over
// flatpak. The previous os-release lookup returned the first provider matching
// the distro, and flatpak/snap/nix/cargo all claim the "linux" wildcard, so an
// unrecognised Arch system ended up defaulting to flatpak.
func TestLinuxPreferredManagers_UnlistedArchDerivativePrefersAURHelper(t *testing.T) {
	osInfo := &types.OSInfo{}
	osInfo.System.OSFamily = "prismlinux"
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"flatpak": pm("flatpak"),
		"pacman":  pm("pacman"),
		"yay":     pm("yay"),
	}

	setDefaultManager(osInfo, linuxPreferredManagers(osInfo))

	if got := osInfo.PackageManager.Default.Name; got != "yay" {
		t.Errorf("default = %q, want yay (AUR helper ahead of pacman and flatpak)", got)
	}
}

// Every family that names native managers must name ones that exist as providers,
// otherwise the preference list silently does nothing.
func TestNativeManagers_ReferenceRealProviders(t *testing.T) {
	loaded := mustLoadEmbedded(t)

	for family, managers := range nativeManagers {
		for _, name := range managers {
			if _, ok := loaded[name]; !ok {
				t.Errorf("family %q prefers %q, which is not an embedded provider", family, name)
			}
		}
	}

	for manager, family := range managerFamily {
		if _, ok := loaded[manager]; !ok {
			t.Errorf("managerFamily maps %q, which is not an embedded provider", manager)
		}
		if _, ok := nativeManagers[family]; !ok {
			t.Errorf("managerFamily maps %q to family %q, which has no native managers", manager, family)
		}
	}
}
