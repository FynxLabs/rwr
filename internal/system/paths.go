package system

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/log/v2"
)

// AddCommonPaths appends common tool directories (/usr/local/bin, Homebrew, Cargo,
// Go, Flatpak, Snap, etc.) to the current PATH if they exist on disk.
func AddCommonPaths() string {
	var paths []string
	existingPath := os.Getenv("PATH")
	if existingPath != "" {
		paths = append(paths, existingPath)
	}

	var commonPaths []string

	switch runtime.GOOS {
	case "windows":
		commonPaths = windowsCommonPaths(os.Getenv)
	default: // Unix-like systems (macOS, Linux)
		currentUser, err := user.Current()
		if err != nil {
			log.Warnf("Error getting current user: %v", err)
		} else {
			homeDir := currentUser.HomeDir
			commonPaths = []string{
				// System paths first (highest priority)
				"/usr/bin",        // Common system path
				"/bin",            // Common system path
				"/usr/sbin",       // Common system path
				"/sbin",           // Common system path
				"/usr/local/bin",  // Common system path
				"/usr/local/sbin", // Common system path

				// User's local system paths (high priority)
				filepath.Join(homeDir, ".local/bin"), // User's local binaries

				// Language-specific user paths (medium-high priority)
				filepath.Join(homeDir, ".cargo/bin"), // User's Cargo binaries
				filepath.Join(homeDir, "go/bin"),     // User's Go binaries

				// System-wide language paths (medium priority)
				"/usr/local/go/bin",    // System Go binaries
				"/usr/local/cargo/bin", // System Cargo binaries

				// Package manager paths (medium-low priority)
				"/nix/var/nix/profiles/default/bin", // Nix
				"/snap/bin",                         // Snap packages
				"/var/lib/flatpak/exports/bin",      // Flatpak

				// Homebrew paths last (lowest priority)
				"/opt/homebrew/bin",               // macOS Homebrew
				"/opt/homebrew/sbin",              // macOS Homebrew sbin
				"/home/linuxbrew/.linuxbrew/bin",  // Linuxbrew
				"/home/linuxbrew/.linuxbrew/sbin", // Linuxbrew sbin
			}
		}
	}

	for _, p := range commonPaths {
		path, err := filepath.EvalSymlinks(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		} else if os.IsNotExist(err) {
			log.Debugf("Path %s does not exist", path)
		} else {
			continue
		}
	}

	return strings.Join(paths, string(os.PathListSeparator))
}

// windowsCommonPaths builds the Windows tool directories, resolving each
// %VARIABLE% through getenv and dropping entries whose variable is unset.
//
// The variables have to be expanded here: Go does not interpret cmd-style
// "%USERPROFILE%\..." strings, so the literal paths never resolved and every
// Windows entry was silently skipped - including scoop's shims directory, which
// an elevated shell often does not inherit in PATH, so scoop went undetected.
//
// getenv is a parameter so the expansion can be tested off Windows.
func windowsCommonPaths(getenv func(string) string) []string {
	entries := []struct {
		variable string
		rest     string
	}{
		{"USERPROFILE", `AppData\Local\Microsoft\WindowsApps`}, // Windows Store apps
		{"USERPROFILE", `scoop\shims`},                         // Scoop package manager
		{"ProgramFiles", `Git\bin`},                            // Git
		{"ProgramFiles", `Go\bin`},                             // Go
		{"ProgramFiles", `nodejs`},                             // Node.js
		{"USERPROFILE", `.cargo\bin`},                          // Cargo (Rust package manager)
	}

	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		base := getenv(e.variable)
		if base == "" {
			log.Debugf("Skipping %s: %%%s%% is not set", e.rest, e.variable)
			continue
		}
		paths = append(paths, filepath.Join(base, e.rest))
	}
	return paths
}

// SetPaths updates the PATH environment variable by appending common tool directories.
func SetPaths() error {
	newPath := AddCommonPaths()
	switch runtime.GOOS {
	case "windows":
		return os.Setenv("Path", newPath)
	default:
		return os.Setenv("PATH", newPath)
	}
}
