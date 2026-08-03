package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
)

// FindManifest returns the manifest file at a repo root, in whichever
// registered format it is written, or "" when the root has none.
func FindManifest(root string) string {
	for _, name := range CandidateFilenames("manifest") {
		candidate := filepath.Join(root, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// LoadManifest strictly decodes a manifest and refuses entries whose init
// path escapes the repo root — the manifest is untrusted input, and its init
// paths are about to be resolved and executed.
func LoadManifest(path string) (*types.Manifest, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- operator-supplied repo root
	if err != nil {
		return nil, fmt.Errorf("error reading manifest %s: %w", path, err)
	}
	format, err := FormatForPath(path)
	if err != nil {
		return nil, err
	}

	var manifest types.Manifest
	if err := UnmarshalBlueprint(data, format, &manifest); err != nil {
		return nil, fmt.Errorf("error decoding manifest %s: %w", path, err)
	}

	root := filepath.Dir(path)
	for i, entry := range manifest.Configurations {
		if entry.Name == "" {
			return nil, fmt.Errorf("manifest %s: configurations[%d] has no name", path, i)
		}
		if entry.Init == "" {
			return nil, fmt.Errorf("manifest %s: configuration %q has no init path", path, entry.Name)
		}
		resolved := filepath.Clean(filepath.Join(root, entry.Init))
		rel, relErr := filepath.Rel(root, resolved)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(entry.Init) {
			return nil, fmt.Errorf("manifest %s: configuration %q init path %q resolves outside the repository", path, entry.Name, entry.Init)
		}
	}
	return &manifest, nil
}

// MatchManifest filters entries against the detected system. Empty matcher
// fields match anything; every declared field must agree.
func MatchManifest(manifest *types.Manifest, osInfo *types.OSInfo) []types.ManifestEntry {
	var matched []types.ManifestEntry
	for _, entry := range manifest.Configurations {
		if entry.OS != "" && !strings.EqualFold(entry.OS, osInfo.System.OS) {
			continue
		}
		if entry.Distro != "" && !strings.EqualFold(entry.Distro, osInfo.System.OSFamily) {
			continue
		}
		if entry.Family != "" && !distroInFamily(osInfo.System.OSFamily, entry.Family) {
			continue
		}
		if entry.Arch != "" && !strings.EqualFold(entry.Arch, osInfo.System.OSArch) {
			continue
		}
		matched = append(matched, entry)
	}
	return matched
}

// distroInFamily is a matcher-level family check (arch covers manjaro etc.).
// The provider layer has its own richer notion; matchers only need the
// operator's own vocabulary to be honored literally plus case folding.
func distroInFamily(distro, family string) bool {
	if strings.EqualFold(distro, family) {
		return true
	}
	families := map[string][]string{
		"arch":   {"arch", "manjaro", "endeavouros", "cachyos", "garuda", "archcraft"},
		"debian": {"debian", "ubuntu", "pop", "linuxmint", "elementary", "raspbian"},
		"rhel":   {"rhel", "fedora", "centos", "rocky", "almalinux"},
		"suse":   {"suse", "opensuse", "opensuse-leap", "opensuse-tumbleweed"},
	}
	for _, member := range families[strings.ToLower(family)] {
		if strings.EqualFold(distro, member) {
			return true
		}
	}
	return false
}
