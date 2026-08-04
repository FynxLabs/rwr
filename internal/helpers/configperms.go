package helpers

import (
	"fmt"
	"os"

	"charm.land/log/v2"
)

// Permissions for rwr's own config tree.
//
// The config file holds a GitHub token and a base64-encoded SSH private key, so
// it is owner-only. These were previously created with os.ModePerm (0777) and
// written at 0644, leaving credentials world-readable - and the directory
// world-writable on any account with a permissive umask.
const (
	ConfigDirPerm  os.FileMode = 0o700
	ConfigFilePerm os.FileMode = 0o600
)

// EnsureConfigDir creates a directory for rwr's config or state at owner-only
// permissions, tightening an existing directory that is more permissive.
func EnsureConfigDir(path string) error {
	if err := os.MkdirAll(path, ConfigDirPerm); err != nil {
		return fmt.Errorf("error creating directory %s: %w", path, err)
	}
	return tighten(path, ConfigDirPerm)
}

// SecureConfigFile restricts a written config file to owner-only.
//
// Call it after viper.WriteConfig or SafeWriteConfig: viper writes at 0644 and
// offers no way to ask for anything narrower.
func SecureConfigFile(path string) error {
	return tighten(path, ConfigFilePerm)
}

// PrecreateSecureConfigFile makes sure path exists at owner-only permissions
// BEFORE viper writes credentials into it. Writing first and tightening after
// leaves a window in which the token is world-readable; viper truncates an
// existing file in place, so a file pre-created at 0600 keeps its mode.
func PrecreateSecureConfigFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, ConfigFilePerm) // #nosec G304 -- rwr's own config path
	if err != nil {
		return fmt.Errorf("error creating %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", path, err)
	}
	// An already-existing file keeps whatever mode it had; narrow it now
	// rather than after the credentials have landed in it.
	return tighten(path, ConfigFilePerm)
}

// tighten clears any permission bits beyond want. It never widens: a caller who
// has deliberately restricted something further keeps their setting.
func tighten(path string, want os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error inspecting %s: %w", path, err)
	}

	current := info.Mode().Perm()
	if current&^want == 0 {
		return nil
	}

	narrowed := current & want
	log.Debugf("Tightening permissions on %s from %#o to %#o", path, current, narrowed)
	if err := os.Chmod(path, narrowed); err != nil {
		return fmt.Errorf("error setting permissions on %s: %w", path, err)
	}
	return nil
}
