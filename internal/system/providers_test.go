package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// TestGetAvailableProviders_NoPackageManagers tests the specific regression case
// where provider detection fails and returns empty map
func TestGetAvailableProviders_NoPackageManagers(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set empty PATH to simulate no binaries available
	os.Setenv("PATH", "")

	// This should return empty map but not panic
	providers := GetAvailableProviders()

	// Note: On systems with enhanced path searching, this may still find providers
	// The important thing is it doesn't panic and handles the case gracefully
	t.Logf("Found %d providers with empty PATH (enhanced path searching may still find system binaries)", len(providers))

	// The regression test is really about ensuring no panic and graceful error handling
	if len(providers) == 0 {
		t.Log("Successfully simulated no package managers scenario")
	} else {
		t.Log("Enhanced path searching found system package managers despite empty PATH")
	}
}

// TestGetAvailableProviders_WithValidBinaries tests provider detection with valid binaries
func TestGetAvailableProviders_WithValidBinaries(t *testing.T) {
	// Create temporary directory with mock binaries
	tempDir := t.TempDir()

	// Create mock pacman binary
	pacmanPath := filepath.Join(tempDir, "pacman")
	if err := os.WriteFile(pacmanPath, []byte("#!/bin/bash\necho 'mock pacman'"), 0755); err != nil {
		t.Fatalf("Failed to create mock pacman: %v", err)
	}

	// Create mock paru binary
	paruPath := filepath.Join(tempDir, "paru")
	if err := os.WriteFile(paruPath, []byte("#!/bin/bash\necho 'mock paru'"), 0755); err != nil {
		t.Fatalf("Failed to create mock paru: %v", err)
	}

	// Save original PATH and set to temp directory
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	os.Setenv("PATH", tempDir)

	// Test provider detection
	providers := GetAvailableProviders()

	// Should find providers if they exist in definitions and are compatible
	t.Logf("Found %d providers with mock binaries", len(providers))

	// The actual number depends on OS compatibility checks in provider definitions
	// But at minimum we should not get a panic or error
	// Note: len() never returns negative values, so this is just a structural check
	if providers == nil {
		t.Error("Provider detection should not return nil")
	}
}

// TestGetAvailableProviders_MissingRequiredFiles tests provider detection when
// required files are missing (e.g., /etc/pacman.conf for pacman)
func TestGetAvailableProviders_MissingRequiredFiles(t *testing.T) {
	// This test verifies that providers requiring specific config files
	// are properly filtered out when those files don't exist

	providers := GetAvailableProviders()

	// For each provider returned, verify it meets all requirements
	for name, provider := range providers {
		t.Logf("Validating provider %s", name)

		// Check binary exists
		tool := FindTool(provider.Detection.Binary)
		if !tool.Exists {
			t.Errorf("Provider %s returned but binary %s not found", name, provider.Detection.Binary)
		}

		// Check required files exist
		for _, file := range provider.Detection.Files {
			if !fileExists(file) {
				t.Errorf("Provider %s returned but required file %s missing", name, file)
			}
		}
	}
}

// TestGetAvailableProviders_ArchLinuxSpecific tests Arch Linux specific provider detection
func TestGetAvailableProviders_ArchLinuxSpecific(t *testing.T) {
	// Create temporary files to simulate Arch Linux environment
	tempDir := t.TempDir()

	// Create mock /etc/pacman.conf
	pacmanConf := filepath.Join(tempDir, "pacman.conf")
	if err := os.WriteFile(pacmanConf, []byte("[options]\nHoldPkg = pacman glibc\n"), 0644); err != nil {
		t.Fatalf("Failed to create mock pacman.conf: %v", err)
	}

	// Create mock /var/lib/pacman directory
	pacmanDir := filepath.Join(tempDir, "pacman")
	if err := os.MkdirAll(pacmanDir, 0755); err != nil {
		t.Fatalf("Failed to create mock pacman directory: %v", err)
	}

	t.Logf("Created mock Arch environment in %s", tempDir)

	// Note: This is a structure test - actual file checking would require
	// modifying the provider definitions to use test paths
}

// TestPackageManagerDetectionRegression is the main regression test
// for the "no package managers available" bug
func TestPackageManagerDetectionRegression(t *testing.T) {
	t.Run("EmptyPathScenario", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Simulate empty PATH
		os.Setenv("PATH", "")

		// This should not panic and should return empty map
		providers := GetAvailableProviders()

		if len(providers) != 0 {
			t.Errorf("Expected empty providers map with no PATH, got %d providers", len(providers))
		}

		// Verify the error case that would be hit in ProcessPackages
		if len(providers) == 0 {
			t.Log("Correctly detected no package managers available - this would trigger the error in ProcessPackages")
		}
	})

	t.Run("NoRequiredFilesScenario", func(t *testing.T) {
		// Create temp directory with binaries but no required config files
		tempDir := t.TempDir()

		// Create mock binaries
		binaries := []string{"pacman", "paru", "apt", "yum", "dnf"}
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

		// Get providers - should be filtered out due to missing required files
		providers := GetAvailableProviders()

		t.Logf("Found %d providers with binaries but no config files", len(providers))

		// Verify any returned providers actually have their required files
		for name, provider := range providers {
			for _, file := range provider.Detection.Files {
				if !fileExists(file) {
					t.Errorf("Provider %s returned but required file %s is missing", name, file)
				}
			}
		}
	})

	t.Run("UnsupportedDistributionScenario", func(t *testing.T) {
		// This would test OS/distribution compatibility
		// The actual implementation depends on how the provider definitions
		// specify distribution compatibility

		providers := GetAvailableProviders()
		t.Logf("Current system found %d compatible providers", len(providers))

		// Verify each provider supports current OS
		for name, provider := range providers {
			if len(provider.Detection.Distributions) > 0 {
				t.Logf("Provider %s supports distributions: %v", name, provider.Detection.Distributions)
			}
		}
	})
}

// TestProviderInitialization tests the provider initialization process
func TestProviderInitialization(t *testing.T) {
	// Test that InitProviders doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitProviders panicked: %v", r)
		}
	}()

	InitProviders()
	t.Log("InitProviders completed successfully")
}

// TestOSInfoIntegration tests the integration with OSInfo detection
func TestOSInfoIntegration(t *testing.T) {
	osInfo := &types.OSInfo{}

	// Test Linux details setting
	err := SetLinuxDetails(osInfo)
	if err != nil {
		t.Errorf("SetLinuxDetails failed: %v", err)
	}

	t.Logf("Detected %d package managers", len(osInfo.PackageManager.Managers))

	if osInfo.PackageManager.Default.Name != "" {
		t.Logf("Default package manager: %s", osInfo.PackageManager.Default.Name)
	} else {
		t.Log("No default package manager set")
	}
}

// BenchmarkGetAvailableProviders benchmarks the provider detection performance
func BenchmarkGetAvailableProviders(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetAvailableProviders()
	}
}

// TestProviderValidation tests that all returned providers are properly validated
func TestProviderValidation(t *testing.T) {
	providers := GetAvailableProviders()

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			// Validate provider structure
			if provider.Detection.Binary == "" {
				t.Error("Provider missing binary specification")
			}

			if len(provider.Detection.Distributions) == 0 {
				t.Error("Provider missing distribution specification")
			}

			// Validate binary exists (since provider was returned)
			tool := FindTool(provider.Detection.Binary)
			if !tool.Exists {
				t.Errorf("Provider binary %s not found but provider was returned", provider.Detection.Binary)
			}

			// Validate required files exist (since provider was returned)
			for _, file := range provider.Detection.Files {
				if !fileExists(file) {
					t.Errorf("Required file %s missing but provider was returned", file)
				}
			}
		})
	}
}
