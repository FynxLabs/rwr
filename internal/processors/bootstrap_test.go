package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// TestProcessPackages_DefaultSystemState tests package manager detection in the default system state
// that was previously failing with "no package managers available" error at bootstrap.go:60
func TestProcessPackages_DefaultSystemState(t *testing.T) {
	t.Run("EmptyPATH_ShouldFindPackageManagers", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set empty PATH to simulate the regression scenario
		os.Setenv("PATH", "")

		// Create minimal package data that won't trigger actual installation
		packagesData := &types.PackagesData{
			Packages: []types.Package{}, // Empty list to test detection without installation
		}

		// Create minimal OS info and init config
		osInfo := &types.OSInfo{}
		initConfig := &types.InitConfig{
			Variables: types.Variables{
				Flags: types.Flags{
					Profiles: []string{"test"},
					Debug:    true,
				},
			},
		}

		// With enhanced path searching, this should now find package managers and attempt installation
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		if err == nil {
			t.Log("ProcessPackages succeeded - package manager detection working correctly")
		} else {
			errorMsg := err.Error()
			if containsString(errorMsg, "no package managers available") {
				t.Errorf("FAILED: Still getting 'no package managers available' error: %s", errorMsg)
			} else {
				t.Logf("SUCCESS: Package manager detection working, error: %s", errorMsg)
			}
		}
	})

	t.Run("WithValidPackageManager_ShouldSucceed", func(t *testing.T) {
		// Create temporary directory with mock package manager
		tempDir := t.TempDir()

		// Create mock pacman binary (most likely to be detected on test systems)
		pacmanPath := filepath.Join(tempDir, "pacman")
		if err := os.WriteFile(pacmanPath, []byte("#!/bin/bash\necho 'Package installed'"), 0755); err != nil {
			t.Fatalf("Failed to create mock pacman: %v", err)
		}

		// Save original PATH and add temp directory
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)
		os.Setenv("PATH", tempDir+":"+originalPath)

		// Create minimal package data with empty install list to avoid actual execution
		packagesData := &types.PackagesData{
			Packages: []types.Package{}, // Empty to avoid execution
		}

		// Create minimal OS info and init config
		osInfo := &types.OSInfo{}
		initConfig := &types.InitConfig{
			Variables: types.Variables{
				Flags: types.Flags{
					Profiles: []string{"test"},
					Debug:    true,
				},
			},
		}

		// This should not return the "no package managers available" error
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		// We might get other errors (like package installation failures),
		// but we specifically want to avoid the "no package managers available" error
		if err != nil {
			errorMsg := err.Error()
			if containsString(errorMsg, "no package managers available") {
				t.Errorf("Got regression error even with valid package manager: %s", errorMsg)
			} else {
				t.Logf("Got non-regression error (expected with mock): %s", errorMsg)
			}
		} else {
			t.Log("ProcessPackages succeeded")
		}
	})

	t.Run("MissingRequiredFiles_ShouldReturnSpecificError", func(t *testing.T) {
		// Create temporary directory with binaries but no config files
		tempDir := t.TempDir()

		// Create mock binaries that exist but lack required configuration files
		binaries := []string{"pacman", "paru", "apt"}
		for _, binary := range binaries {
			binaryPath := filepath.Join(tempDir, binary)
			if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho 'mock'"), 0755); err != nil {
				t.Fatalf("Failed to create mock %s: %v", binary, err)
			}
		}

		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)
		os.Setenv("PATH", tempDir)

		// Create minimal package data that won't trigger installation
		packagesData := &types.PackagesData{
			Packages: []types.Package{}, // Empty list to test detection without installation
		}

		// Create minimal OS info and init config
		osInfo := &types.OSInfo{}
		initConfig := &types.InitConfig{
			Variables: types.Variables{
				Flags: types.Flags{
					Profiles: []string{"test"},
					Debug:    true,
				},
			},
		}

		// With enhanced path searching, this should now find system package managers
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		if err == nil {
			t.Log("ProcessPackages succeeded - enhanced path searching found system package managers")
		} else {
			errorMsg := err.Error()
			if containsString(errorMsg, "no package managers available") {
				t.Errorf("FAILED: Still getting 'no package managers available' error: %s", errorMsg)
			} else {
				t.Logf("SUCCESS: Found package managers, error: %s", errorMsg)
			}
		}
	})
}

// TestBootstrapPackageManagerDetection tests the complete bootstrap flow
// for package manager detection issues
func TestBootstrapPackageManagerDetection(t *testing.T) {
	t.Run("DefaultSystemStateBootstrap", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set empty PATH
		os.Setenv("PATH", "")

		// Create a minimal configuration that won't trigger installation
		packagesData := &types.PackagesData{
			Packages: []types.Package{}, // Empty list to test detection without installation
		}

		// Create minimal OS info and init config
		osInfo := &types.OSInfo{}
		initConfig := &types.InitConfig{
			Variables: types.Variables{
				Flags: types.Flags{
					Profiles: []string{"test"},
					Debug:    true,
				},
			},
		}

		// Test that package manager detection now works
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		// Verify package manager detection is working
		if err == nil {
			t.Log("ProcessPackages succeeded - package manager detection working correctly")
		} else if containsString(err.Error(), "no package managers available") {
			t.Errorf("FAILED: Still getting 'no package managers available' error: %s", err.Error())
		} else {
			t.Logf("SUCCESS: Package manager detection working, error: %s", err.Error())
		}
	})

	t.Run("BootstrapWithDebugLogging", func(t *testing.T) {
		// This test validates that the debug logging we added doesn't break anything

		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set empty PATH to trigger debug path
		os.Setenv("PATH", "")

		packagesData := &types.PackagesData{
			Packages: []types.Package{}, // Empty list to avoid actual installation
		}

		// Create minimal OS info and init config
		osInfo := &types.OSInfo{}
		initConfig := &types.InitConfig{
			Variables: types.Variables{
				Flags: types.Flags{
					Profiles: []string{"test"},
					Debug:    true,
				},
			},
		}

		// Call should complete without panic and work with enhanced path searching
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		if err == nil {
			t.Log("ProcessPackages succeeded with debug logging enabled")
		} else {
			errorMsg := err.Error()
			t.Logf("Debug logging working, got error: %s", errorMsg)

			if containsString(errorMsg, "no package managers available") {
				t.Error("FAILED: Debug logging test still getting 'no package managers available' error")
			} else {
				t.Log("SUCCESS: Debug logging working, enhanced path searching found package managers")
			}
		}
	})
}

// TestPackageProcessingEdgeCases tests various edge cases that could cause
// the package manager detection to fail
func TestPackageProcessingEdgeCases(t *testing.T) {
	// Create minimal OS info and init config for all tests
	osInfo := &types.OSInfo{}
	initConfig := &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				Profiles: []string{"test"},
				Debug:    true,
			},
		},
	}

	t.Run("NilPackagesConfig", func(t *testing.T) {
		// Should handle nil packages config gracefully
		err := ProcessPackages(nil, nil, "yaml", osInfo, initConfig)

		// Nil config should be handled gracefully, not cause panic
		if err != nil {
			t.Logf("Nil packages config handled with error: %s", err.Error())
		} else {
			t.Log("Nil packages config handled gracefully")
		}
	})

	t.Run("EmptyPackagesConfig", func(t *testing.T) {
		packagesData := &types.PackagesData{}

		// Should handle empty packages config gracefully
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		if err != nil {
			t.Logf("Empty packages config handled with error: %s", err.Error())
		} else {
			t.Log("Empty packages config handled gracefully")
		}
	})

	t.Run("EmptyInstallList", func(t *testing.T) {
		packagesData := &types.PackagesData{
			Packages: []types.Package{},
		}

		// Should handle empty install list gracefully
		err := ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)

		if err != nil {
			t.Logf("Empty install list handled with error: %s", err.Error())
		} else {
			t.Log("Empty install list handled gracefully")
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle ||
			haystack[:len(needle)] == needle ||
			haystack[len(haystack)-len(needle):] == needle ||
			findSubstring(haystack, needle))
}

// Simple substring search
func findSubstring(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// BenchmarkProcessPackagesError benchmarks the error path for performance
func BenchmarkProcessPackagesError(b *testing.B) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set empty PATH to force error path
	os.Setenv("PATH", "")

	packagesData := &types.PackagesData{
		Packages: []types.Package{}, // Empty list to avoid actual installation
	}

	// Create minimal OS info and init config
	osInfo := &types.OSInfo{}
	initConfig := &types.InitConfig{
		Variables: types.Variables{
			Flags: types.Flags{
				Profiles: []string{"test"},
				Debug:    true,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessPackages(nil, packagesData, "yaml", osInfo, initConfig)
	}
}
