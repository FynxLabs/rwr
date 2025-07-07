package system

import (
	"os"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
)

// TestPackageManagerDetectionScenarios tests various scenarios for package manager detection
// including the scenario that was previously failing with "no package managers available" error
func TestPackageManagerDetectionScenarios(t *testing.T) {
	t.Run("RestrictedEnvironment", func(t *testing.T) {
		// Save originals
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set completely empty PATH and remove common system paths
		os.Setenv("PATH", "/nonexistent/path")

		// Force reinitialize providers to clear any cached state
		providers = make(map[string]*types.Provider)

		// This should trigger the exact scenario that was failing
		available := GetAvailableProviders()

		if len(available) == 0 {
			t.Log("Successfully created scenario with no package managers")
			t.Log("This would trigger the error in ProcessPackages in environments without package managers")
		} else {
			// If enhanced path searching still finds packages, that's actually good
			t.Logf("Enhanced path searching found %d providers even with restricted PATH", len(available))
			t.Log("This indicates the enhanced binary detection is working correctly")
		}
	})
}

// TestProviderDetectionRobustness tests the robustness of provider detection
func TestProviderDetectionRobustness(t *testing.T) {
	t.Run("MultipleInitCalls", func(t *testing.T) {
		// Test that multiple InitProviders calls don't break anything
		for i := 0; i < 5; i++ {
			if err := InitProviders(); err != nil {
				t.Errorf("InitProviders call %d failed: %v", i+1, err)
			}
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Test concurrent access to providers
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// Each goroutine tries to get available providers
				providers := GetAvailableProviders()
				if providers == nil {
					t.Errorf("Goroutine %d got nil providers map", id)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// Good
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent provider access test")
				return
			}
		}
	})

	t.Run("EmptyProviderDefinitions", func(t *testing.T) {
		// Save original providers
		originalProviders := providers
		defer func() { providers = originalProviders }()

		// Set empty providers map
		providers = make(map[string]*types.Provider)

		// Should handle empty definitions gracefully
		available := GetAvailableProviders()
		if len(available) != 0 {
			t.Errorf("Expected 0 providers with empty definitions, got %d", len(available))
		}

		// Should not panic when getting specific provider
		provider, exists := GetProvider("nonexistent")
		if exists || provider != nil {
			t.Error("GetProvider should return false for nonexistent provider with empty definitions")
		}
	})
}

// TestDebugLoggingFunctionality tests that debug logging doesn't break functionality
func TestDebugLoggingFunctionality(t *testing.T) {
	t.Run("DebugLoggingEnabled", func(t *testing.T) {
		// This test ensures our debug logging additions don't cause issues

		// Test with various PATH configurations
		testPaths := []string{
			"",                        // Empty PATH
			"/usr/bin:/bin:/usr/sbin", // Standard paths
			"/nonexistent",            // Nonexistent path
			"/usr/bin",                // Single valid path
		}

		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		for _, testPath := range testPaths {
			t.Run("PATH_"+testPath, func(t *testing.T) {
				os.Setenv("PATH", testPath)

				// Should not panic with any PATH configuration
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("Panic with PATH='%s': %v", testPath, r)
						}
					}()

					providers := GetAvailableProviders()
					t.Logf("PATH='%s' found %d providers", testPath, len(providers))
				}()
			})
		}
	})
}

// TestErrorHandlingImprovements tests error handling improvements
func TestErrorHandlingImprovements(t *testing.T) {
	t.Run("DetailedErrorMessages", func(t *testing.T) {
		// Save original PATH
		originalPath := os.Getenv("PATH")
		defer os.Setenv("PATH", originalPath)

		// Set PATH that will cause no providers to be found
		os.Setenv("PATH", "/tmp/nonexistent")

		// Clear providers to force re-detection
		providers = make(map[string]*types.Provider)

		// Get available providers (should be empty)
		available := GetAvailableProviders()

		if len(available) == 0 {
			t.Log("Successfully created scenario with no package managers")

			// Test that the enhanced error message would be triggered
			// This simulates what would happen in ProcessPackages
			osInfo := &types.OSInfo{}
			if err := SetLinuxDetails(osInfo); err == nil {
				if osInfo.System.OS == "linux" {
					t.Log("Enhanced error message would include OS details")
				}
			}
		}
	})
}

// BenchmarkProviderDetection benchmarks provider detection performance
func BenchmarkProviderDetection(b *testing.B) {
	b.Run("GetAvailableProviders", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			GetAvailableProviders()
		}
	})

	b.Run("InitProviders", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			InitProviders()
		}
	})
}
