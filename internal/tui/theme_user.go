package tui

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// LoadUserTheme reads <configDir>/themes/<name>.toml. A missing or broken
// file returns false and the caller falls back to the built-ins.
func LoadUserTheme(configDir, name string) (Theme, bool) {
	if configDir == "" || name == "" {
		return Theme{}, false
	}
	path := filepath.Join(configDir, "themes", name+".toml")
	data, err := os.ReadFile(path) // #nosec G304 -- rwr's own config dir
	if err != nil {
		return Theme{}, false
	}
	var fields map[string]string
	if _, err := toml.Decode(string(data), &fields); err != nil {
		return Theme{}, false
	}
	theme := rwrTheme
	theme.Name = name
	set := func(dst *string, key string) {
		if v, ok := fields[key]; ok && v != "" {
			*dst = v
		}
	}
	set(&theme.Text, "text")
	set(&theme.Subtext, "subtext")
	set(&theme.Muted, "muted")
	set(&theme.Dim, "dim")
	set(&theme.Accent, "accent")
	set(&theme.Success, "success")
	set(&theme.Danger, "danger")
	set(&theme.Warning, "warning")
	set(&theme.Info, "info")
	set(&theme.Stdout, "stdout")
	set(&theme.Stderr, "stderr")
	if fields["glyphs"] == "ascii" {
		theme.Glyphs = asciiGlyphs
	}
	return theme, true
}
