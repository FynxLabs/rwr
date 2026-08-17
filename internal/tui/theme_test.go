package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// mustTheme resolves a built-in by name for tests; user-theme lookup is
// disabled by the empty config dir.
func mustTheme(name string) Theme {
	theme, _ := ResolveTheme("", name, "", false, false)
	return theme
}

func mustThemeASCII(name string) Theme {
	theme, _ := ResolveTheme("", name, "", true, false)
	return theme
}

func TestResolveThemeBuiltin(t *testing.T) {
	theme, unknown := ResolveTheme("", "nord", "", false, false)
	if theme.Name != "nord" || unknown != "" {
		t.Fatalf("got theme %q, unknown %q", theme.Name, unknown)
	}
}

func TestResolveThemeUnknownFallsBackAndReports(t *testing.T) {
	theme, unknown := ResolveTheme(t.TempDir(), "no-such-theme", "", false, false)
	if theme.Name != "rwr" {
		t.Fatalf("fallback theme = %q, want rwr", theme.Name)
	}
	if unknown != "no-such-theme" {
		t.Fatalf("unknown = %q, want the requested name", unknown)
	}
}

func TestResolveThemeEmptyNameIsNotUnknown(t *testing.T) {
	theme, unknown := ResolveTheme("", "", "", false, false)
	if theme.Name != "rwr" || unknown != "" {
		t.Fatalf("got theme %q, unknown %q", theme.Name, unknown)
	}
}

func TestResolveThemeUserFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	toml := "accent = \"#123456\"\nmodal = \"#654321\"\n"
	if err := os.WriteFile(filepath.Join(dir, "themes", "mine.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	theme, unknown := ResolveTheme(dir, "mine", "", false, false)
	if unknown != "" {
		t.Fatalf("unknown = %q for an existing user theme", unknown)
	}
	if theme.Name != "mine" || theme.Accent != "#123456" || theme.Modal != "#654321" {
		t.Fatalf("user theme not loaded: name %q accent %q modal %q", theme.Name, theme.Accent, theme.Modal)
	}
	// Unset fields inherit the rwr defaults.
	if theme.Success != rwrTheme.Success {
		t.Fatalf("unset field did not inherit: %q", theme.Success)
	}
}

func TestResolveThemeUserOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "themes", "nord.toml"), []byte("accent = \"#000001\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	theme, _ := ResolveTheme(dir, "nord", "", false, false)
	if theme.Accent != "#000001" {
		t.Fatalf("user nord did not override built-in: accent %q", theme.Accent)
	}
}
