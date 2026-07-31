package helpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The config file holds a GitHub token and a base64 SSH private key. It was
// created inside a directory made with os.ModePerm (0777) and written by viper at
// 0644, so on any account with a permissive umask the credentials were
// world-readable and the directory world-writable.
func TestEnsureConfigDir_IsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}

	dir := filepath.Join(t.TempDir(), "rwr")

	if err := EnsureConfigDir(dir); err != nil {
		t.Fatalf("EnsureConfigDir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != ConfigDirPerm {
		t.Errorf("directory mode = %#o, want %#o", perm, ConfigDirPerm)
	}
}

// An existing world-readable directory from an earlier version must be tightened,
// not left as it is.
func TestEnsureConfigDir_TightensExistingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}

	dir := filepath.Join(t.TempDir(), "rwr")
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if err := EnsureConfigDir(dir); err != nil {
		t.Fatalf("EnsureConfigDir: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("directory mode = %#o, want no group or other access", perm)
	}
}

func TestSecureConfigFile_IsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	// 0644 is what viper writes.
	if err := os.WriteFile(path, []byte("repository:\n  gh_api_token: ghp_x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := SecureConfigFile(path); err != nil {
		t.Fatalf("SecureConfigFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != ConfigFilePerm {
		t.Errorf("file mode = %#o, want %#o", perm, ConfigFilePerm)
	}
}

// Tightening must never widen: an operator who restricted something further
// keeps their setting.
func TestSecureConfigFile_DoesNotWidenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("x: y\n"), 0o400); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := SecureConfigFile(path); err != nil {
		t.Fatalf("SecureConfigFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o400 {
		t.Errorf("file mode = %#o, want it left at 0400", perm)
	}
}
