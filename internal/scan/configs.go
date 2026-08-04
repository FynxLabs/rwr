package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ConfigResult is one discovered config: a known dotfile or a ~/.config
// entry.
type ConfigResult struct {
	Path  string // absolute
	Rel   string // relative to home, the identity a blueprint uses
	Known bool   // in the known-dotfile set (pre-selected by consumers)
}

// knownDotfiles is the set worth pre-selecting: the files people mean when
// they say "my dotfiles".
var knownDotfiles = []string{
	".bashrc", ".bash_profile", ".zshrc", ".zprofile", ".profile",
	".aliases", ".exports", ".functions", ".path",
	".gitconfig", ".gitignore", ".gitignore_global",
	".vimrc", ".tmux.conf", ".inputrc", ".editorconfig",
}

// configNoise names the ~/.config entries that are cache, state, or session
// plumbing - shown only with includeAll. The human selects from what is
// shown, so what is shown must be worth reading.
var configNoise = map[string]bool{
	"pulse": true, "dconf": true, "ibus": true, "gtk-2.0": true,
	"session": true, "systemd": true, "autostart": true, "Trash": true,
	"cache": true, "chromium": true, "google-chrome": true, "BraveSoftware": true,
	"Slack": true, "discord": true, "Electron": true, "Code": true,
	"pipewire-media-session": true, "pipewire": true, "dbus": true,
	"enchant": true, "goa-1.0": true, "evolution": true, "gnome-session": true,
}

// secretShaped reports paths that must never be pre-selected: key material
// travels by keyring or by hand, not by blueprint capture.
func secretShaped(rel string) bool {
	if strings.HasPrefix(rel, ".ssh/") || rel == ".ssh" {
		return true
	}
	for _, marker := range []string{".gnupg", "private", "id_rsa", "id_ed25519", ".netrc", "credentials"} {
		if strings.Contains(strings.ToLower(rel), marker) {
			return true
		}
	}
	return false
}

// SecretShaped is the exported check consumers use to refuse pre-selection
// and warn on explicit selection.
func SecretShaped(rel string) bool { return secretShaped(rel) }

// Configs reports the known dotfiles present in home plus the top-level
// ~/.config entries, minus the noise list unless includeAll is set.
func Configs(home string, includeAll bool) []ConfigResult {
	var results []ConfigResult

	for _, name := range knownDotfiles {
		path := filepath.Join(home, name)
		if _, err := os.Stat(path); err == nil {
			results = append(results, ConfigResult{Path: path, Rel: name, Known: true})
		}
	}

	configDir := filepath.Join(home, ".config")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return results
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		if !includeAll && configNoise[name] {
			continue
		}
		rel := filepath.Join(".config", name)
		if !includeAll && secretShaped(rel) {
			continue
		}
		results = append(results, ConfigResult{
			Path: filepath.Join(configDir, name),
			Rel:  rel,
		})
	}
	return results
}
