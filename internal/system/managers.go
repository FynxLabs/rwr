package system

import (
	"sort"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// collectAvailableManagers records every provider usable on this system in
// osInfo.PackageManager.Managers.
//
// GetAvailableProviders has already applied supportsSystem (exact distro, distro
// family, or the "linux" wildcard), confirmed the binary exists, and stored its
// path in BinPath — so no further filtering belongs here. The per-OS callers used
// to re-filter with a literal string match against the provider's distribution
// list, which excluded every provider that names concrete distros: on Linux that
// meant apt, dnf, pacman, zypper and the AUR helpers never made it into the map,
// leaving cache cleaning and core-package resolution working off a map that held
// only the wildcard providers.
func collectAvailableManagers(osInfo *types.OSInfo) {
	if osInfo.PackageManager.Managers == nil {
		osInfo.PackageManager.Managers = make(map[string]types.PackageManagerInfo)
	}

	for name, prov := range GetAvailableProviders() {
		osInfo.PackageManager.Managers[name] = GetPackageManagerInfo(prov, prov.BinPath)
		log.Debugf("Added package manager: %s", name)
	}
}

// setDefaultManager picks osInfo.PackageManager.Default as the first entry of
// preferred that is actually present, falling back to the alphabetically first
// available manager.
//
// The fallback is sorted rather than "whatever the map yields first": Go
// randomizes map iteration, so the previous version could resolve a different
// default on every run, meaning a package with no explicit package_manager could
// be installed by a different tool each time.
func setDefaultManager(osInfo *types.OSInfo, preferred []string) {
	for _, name := range preferred {
		if pm, ok := osInfo.PackageManager.Managers[name]; ok && pm.Bin != "" {
			osInfo.PackageManager.Default = pm
			log.Infof("Set %s as default package manager", pm.Name)
			return
		}
	}

	names := make([]string, 0, len(osInfo.PackageManager.Managers))
	for name := range osInfo.PackageManager.Managers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if pm := osInfo.PackageManager.Managers[name]; pm.Bin != "" {
			osInfo.PackageManager.Default = pm
			log.Infof("Set %s as default package manager", pm.Name)
			return
		}
	}

	log.Warn("No default package manager set")
}
