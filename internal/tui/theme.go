// Package tui is the Bubble Tea dashboard for interactive runs. Non-TTY
// behavior is untouched: activation falls back to the LogReporter, whose
// output is byte-identical to the pre-TUI stream.
package tui

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	catppuccin "github.com/catppuccin/go"
)

// Glyphs are part of the theme, not a separate mechanism; the ASCII set is a
// base other themes inherit when unicode is unsafe.
type Glyphs struct {
	Done, Failed, Degraded, Unknown, Pending, Skipped string
	StripFilled, StripEmpty                           string
	BarFull, BarEmpty                                 string
	Spinner                                           []string
}

var unicodeGlyphs = Glyphs{
	Done: "✓", Failed: "✗", Degraded: "⚠", Unknown: "?", Pending: "○", Skipped: "–",
	StripFilled: "▰", StripEmpty: "▱",
	BarFull: "█", BarEmpty: "░",
	Spinner: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
}

var asciiGlyphs = Glyphs{
	Done: "+", Failed: "x", Degraded: "!", Unknown: "?", Pending: ".", Skipped: "-",
	StripFilled: "#", StripEmpty: ".",
	BarFull: "#", BarEmpty: "-",
	Spinner: []string{"|", "/", "-", "\\"},
}

// Theme is the eleven color roles plus glyphs. Color is by state, not by
// processor — thirteen distinguishable accents is more than most palettes
// carry. No background is ever painted: the terminal's own is inherited, so
// light-theme and transparent-terminal users keep a working frame.
type Theme struct {
	Name string

	Text, Subtext, Muted, Dim              string
	Accent, Success, Danger, Warning, Info string
	Stdout, Stderr                         string

	Glyphs Glyphs
}

// rwrTheme is the default, built from Go brand colors.
var rwrTheme = Theme{
	Name: "rwr",
	Text: "#FFFFFF", Subtext: "#D0D0D0", Muted: "#8A8A8A", Dim: "#585858",
	Accent: "#00ADD8", Success: "#00A29C", Danger: "#CE3262", Warning: "#FDDD00", Info: "#5DC9E2",
	Stdout: "#B2B2B2", Stderr: "#D78787",
	Glyphs: unicodeGlyphs,
}

// catppuccinTheme maps a catppuccin flavour onto the roles.
func catppuccinTheme(name string, flavour catppuccin.Flavor) Theme {
	return Theme{
		Name: name,
		Text: flavour.Text().Hex, Subtext: flavour.Subtext1().Hex,
		Muted: flavour.Overlay1().Hex, Dim: flavour.Surface2().Hex,
		Accent: flavour.Blue().Hex, Success: flavour.Green().Hex,
		Danger: flavour.Red().Hex, Warning: flavour.Yellow().Hex, Info: flavour.Sky().Hex,
		Stdout: flavour.Subtext0().Hex, Stderr: flavour.Maroon().Hex,
		Glyphs: unicodeGlyphs,
	}
}

// builtinThemes ships embedded; user themes are TOML files in the config dir
// with the same schema (LoadUserTheme), so copying a built-in and editing it
// just works — the providers pattern.
func builtinThemes() map[string]Theme {
	themes := map[string]Theme{
		"rwr":       rwrTheme,
		"latte":     catppuccinTheme("latte", catppuccin.Latte),
		"frappe":    catppuccinTheme("frappe", catppuccin.Frappe),
		"macchiato": catppuccinTheme("macchiato", catppuccin.Macchiato),
		"mocha":     catppuccinTheme("mocha", catppuccin.Mocha),
		"nord": {
			Name: "nord", Text: "#ECEFF4", Subtext: "#E5E9F0", Muted: "#7B88A1", Dim: "#4C566A",
			Accent: "#88C0D0", Success: "#A3BE8C", Danger: "#BF616A", Warning: "#EBCB8B", Info: "#81A1C1",
			Stdout: "#D8DEE9", Stderr: "#D08770", Glyphs: unicodeGlyphs,
		},
		"gruvbox": {
			Name: "gruvbox", Text: "#EBDBB2", Subtext: "#D5C4A1", Muted: "#928374", Dim: "#665C54",
			Accent: "#83A598", Success: "#B8BB26", Danger: "#FB4934", Warning: "#FABD2F", Info: "#8EC07C",
			Stdout: "#BDAE93", Stderr: "#FE8019", Glyphs: unicodeGlyphs,
		},
		"dracula": {
			Name: "dracula", Text: "#F8F8F2", Subtext: "#E6E6E6", Muted: "#6272A4", Dim: "#44475A",
			Accent: "#BD93F9", Success: "#50FA7B", Danger: "#FF5555", Warning: "#F1FA8C", Info: "#8BE9FD",
			Stdout: "#CFCFC2", Stderr: "#FFB86C", Glyphs: unicodeGlyphs,
		},
		"tokyonight": {
			Name: "tokyonight", Text: "#C0CAF5", Subtext: "#A9B1D6", Muted: "#565F89", Dim: "#3B4261",
			Accent: "#7AA2F7", Success: "#9ECE6A", Danger: "#F7768E", Warning: "#E0AF68", Info: "#7DCFFF",
			Stdout: "#A9B1D6", Stderr: "#FF9E64", Glyphs: unicodeGlyphs,
		},
		"rose-pine": {
			Name: "rose-pine", Text: "#E0DEF4", Subtext: "#908CAA", Muted: "#6E6A86", Dim: "#403D52",
			Accent: "#C4A7E7", Success: "#31748F", Danger: "#EB6F92", Warning: "#F6C177", Info: "#9CCFD8",
			Stdout: "#908CAA", Stderr: "#EBBCBA", Glyphs: unicodeGlyphs,
		},
		"solarized": {
			Name: "solarized", Text: "#EEE8D5", Subtext: "#93A1A1", Muted: "#657B83", Dim: "#586E75",
			Accent: "#268BD2", Success: "#859900", Danger: "#DC322F", Warning: "#B58900", Info: "#2AA198",
			Stdout: "#93A1A1", Stderr: "#CB4B16", Glyphs: unicodeGlyphs,
		},
	}
	return themes
}

// unicodeSafe decides glyph capability: UTF-8 locales are safe; on Windows,
// Windows Terminal (WT_SESSION) is safe and conhost is not.
func unicodeSafe() bool {
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	for _, env := range []string{"LC_ALL", "LANG"} {
		if v := os.Getenv(env); strings.Contains(strings.ToUpper(v), "UTF-8") || strings.Contains(strings.ToUpper(v), "UTF8") {
			return true
		}
	}
	return false
}

// ResolveTheme picks the theme: --theme flag, config key, $RWR_THEME, then
// the default. NO_COLOR overrides all of it (colors drop; glyphs stay per
// capability). forceASCII/forceUnicode come from --ascii/--unicode.
// A TOML theme in <configDir>/themes/ overrides a built-in of the same name —
// the providers pattern. unknown names the requested theme when nothing
// matched and the default was substituted, so the caller can warn.
func ResolveTheme(configDir, flagTheme, configTheme string, forceASCII, forceUnicode bool) (resolved Theme, unknown string) {
	name := flagTheme
	if name == "" {
		name = configTheme
	}
	if name == "" {
		name = os.Getenv("RWR_THEME")
	}
	theme, ok := LoadUserTheme(configDir, name)
	if !ok {
		theme, ok = builtinThemes()[name]
	}
	if !ok {
		if name != "" {
			unknown = name
		}
		theme = builtinThemes()["rwr"]
	}

	switch {
	case forceASCII:
		theme.Glyphs = asciiGlyphs
	case forceUnicode:
		theme.Glyphs = unicodeGlyphs
	case !unicodeSafe():
		theme.Glyphs = asciiGlyphs
	}

	if os.Getenv("NO_COLOR") != "" {
		blank := Theme{Name: theme.Name + "+no-color", Glyphs: theme.Glyphs}
		return blank, unknown
	}
	return theme, unknown
}

// style returns a foreground style for a role color; empty color renders
// unstyled (the NO_COLOR degradation: glyphs and dim/bold only).
func style(color string) lipgloss.Style {
	if color == "" {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}
