package system

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
)

// AddCommonPaths checks for common paths and appends them to the existing PATH environment variable.
func AddCommonPaths() string {
	var paths []string
	existingPath := os.Getenv("PATH")
	if existingPath != "" {
		paths = append(paths, existingPath)
	}

	var commonPaths []string

	switch runtime.GOOS {
	case "windows":
		commonPaths = []string{
			"%USERPROFILE%\\AppData\\Local\\Microsoft\\WindowsApps", // Path for Windows Store apps
			"%USERPROFILE%\\scoop\\shims",                           // Path for Scoop package manager
			"%PROGRAMFILES%\\Git\\bin",                              // Path for Git
			"%PROGRAMFILES%\\Go\\bin",                               // Path for Go
			"%PROGRAMFILES%\\nodejs",                                // Path for Node.js
			"%PROGRAMFILES%\\Rust\\.cargo\\bin",                     // Path for Cargo (Rust package manager)
		}
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

// SetPaths sets the PATH environment variable with the common paths appended.
func SetPaths() error {
	newPath := AddCommonPaths()
	switch runtime.GOOS {
	case "windows":
		return os.Setenv("Path", newPath)
	default:
		return os.Setenv("PATH", newPath)
	}
}
