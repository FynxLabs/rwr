package helpers

import (
	"reflect"
	"slices"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// TestIntegration_RealWorldScenario tests a complete real-world configuration
func TestIntegration_RealWorldScenario(t *testing.T) {
	// Create a realistic configuration that mirrors what users might have
	workstationConfig := createWorkstationConfiguration()

	// Test scenario 1: No profiles - only base items
	t.Run("no_profiles_base_only", func(t *testing.T) {
		activeProfiles := []string{}

		filteredPackages := FilterByProfiles(workstationConfig.packages, activeProfiles)
		filteredServices := FilterByProfiles(workstationConfig.services, activeProfiles)
		filteredFiles := FilterByProfiles(workstationConfig.files, activeProfiles)

		// Should get all items (permissive default when no profiles specified)
		expectedPackageCount := 12 // all packages
		expectedServiceCount := 5  // all services
		expectedFileCount := 3     // all files

		if len(filteredPackages) != expectedPackageCount {
			t.Errorf("Expected %d packages, got %d", expectedPackageCount, len(filteredPackages))
		}
		if len(filteredServices) != expectedServiceCount {
			t.Errorf("Expected %d services, got %d", expectedServiceCount, len(filteredServices))
		}
		if len(filteredFiles) != expectedFileCount {
			t.Errorf("Expected %d files, got %d", expectedFileCount, len(filteredFiles))
		}

		// Verify we got all packages (permissive default)
		packageNames := extractPackageNames(filteredPackages)
		expectedAll := []string{"code", "curl", "discord", "docker", "git", "kubectl", "nodejs", "python", "steam", "terraform", "tmux", "vim"}
		slices.Sort(packageNames)
		if !reflect.DeepEqual(packageNames, expectedAll) {
			t.Errorf("All packages = %v, expected %v", packageNames, expectedAll)
		}
	})

	// Test scenario 2: Work profile - base + work items
	t.Run("work_profile", func(t *testing.T) {
		activeProfiles := []string{"work"}

		filteredPackages := FilterByProfiles(workstationConfig.packages, activeProfiles)
		filteredServices := FilterByProfiles(workstationConfig.services, activeProfiles)

		// Should get base + work items
		packageNames := extractPackageNames(filteredPackages)
		expectedPackages := []string{"curl", "docker", "git", "kubectl", "terraform", "tmux", "vim"}
		slices.Sort(packageNames)
		if !reflect.DeepEqual(packageNames, expectedPackages) {
			t.Errorf("Work profile packages = %v, expected %v", packageNames, expectedPackages)
		}

		serviceNames := extractServiceNames(filteredServices)
		expectedServices := []string{"docker", "nginx", "sshd"}
		slices.Sort(serviceNames)
		if !reflect.DeepEqual(serviceNames, expectedServices) {
			t.Errorf("Work profile services = %v, expected %v", serviceNames, expectedServices)
		}
	})

	// Test scenario 3: Multiple profiles - work + gaming
	t.Run("multiple_profiles_work_gaming", func(t *testing.T) {
		activeProfiles := []string{"work", "gaming"}

		filteredPackages := FilterByProfiles(workstationConfig.packages, activeProfiles)
		packageNames := extractPackageNames(filteredPackages)

		// Should get base + work + gaming items
		expectedPackages := []string{"curl", "discord", "docker", "git", "kubectl", "steam", "terraform", "tmux", "vim"}
		slices.Sort(packageNames)
		if !reflect.DeepEqual(packageNames, expectedPackages) {
			t.Errorf("Work+Gaming packages = %v, expected %v", packageNames, expectedPackages)
		}
	})

	// Test scenario 4: "all" profile - everything
	t.Run("all_profile", func(t *testing.T) {
		activeProfiles := []string{"all"}

		filteredPackages := FilterByProfiles(workstationConfig.packages, activeProfiles)

		// Should get everything
		if len(filteredPackages) != len(workstationConfig.packages) {
			t.Errorf("All profile should include all packages: got %d, expected %d",
				len(filteredPackages), len(workstationConfig.packages))
		}
	})
}

// TestIntegration_ProfileDiscovery tests cross-type profile discovery
func TestIntegration_ProfileDiscovery(t *testing.T) {
	config := createWorkstationConfiguration()

	// Discover profiles from each type
	packageProfiles := GetUniqueProfiles(config.packages)
	serviceProfiles := GetUniqueProfiles(config.services)
	fileProfiles := GetUniqueProfiles(config.files)
	userProfiles := GetUniqueProfiles(config.users)

	t.Run("package_profiles", func(t *testing.T) {
		expected := []string{"dev", "gaming", "work"}
		if !reflect.DeepEqual(packageProfiles, expected) {
			t.Errorf("Package profiles = %v, expected %v", packageProfiles, expected)
		}
	})

	t.Run("service_profiles", func(t *testing.T) {
		expected := []string{"dev", "gaming", "work"}
		if !reflect.DeepEqual(serviceProfiles, expected) {
			t.Errorf("Service profiles = %v, expected %v", serviceProfiles, expected)
		}
	})

	t.Run("file_profiles", func(t *testing.T) {
		expected := []string{"dev", "work"}
		if !reflect.DeepEqual(fileProfiles, expected) {
			t.Errorf("File profiles = %v, expected %v", fileProfiles, expected)
		}
	})

	t.Run("user_profiles", func(t *testing.T) {
		expected := []string{"dev"}
		if !reflect.DeepEqual(userProfiles, expected) {
			t.Errorf("User profiles = %v, expected %v", userProfiles, expected)
		}
	})

	// Test collecting all unique profiles across all types
	t.Run("all_unique_profiles", func(t *testing.T) {
		allProfiles := make(map[string]bool)

		for _, profile := range packageProfiles {
			allProfiles[profile] = true
		}
		for _, profile := range serviceProfiles {
			allProfiles[profile] = true
		}
		for _, profile := range fileProfiles {
			allProfiles[profile] = true
		}
		for _, profile := range userProfiles {
			allProfiles[profile] = true
		}

		var uniqueProfiles []string
		for profile := range allProfiles {
			uniqueProfiles = append(uniqueProfiles, profile)
		}
		slices.Sort(uniqueProfiles)

		expected := []string{"dev", "gaming", "work"}
		if !reflect.DeepEqual(uniqueProfiles, expected) {
			t.Errorf("All unique profiles = %v, expected %v", uniqueProfiles, expected)
		}
	})
}

// TestIntegration_ProfileValidation tests validation across real configuration
func TestIntegration_ProfileValidation(t *testing.T) {
	config := createWorkstationConfiguration()

	// Get all available profiles from configuration
	allAvailableProfiles := make(map[string]bool)

	for _, profile := range GetUniqueProfiles(config.packages) {
		allAvailableProfiles[profile] = true
	}
	for _, profile := range GetUniqueProfiles(config.services) {
		allAvailableProfiles[profile] = true
	}
	for _, profile := range GetUniqueProfiles(config.files) {
		allAvailableProfiles[profile] = true
	}
	for _, profile := range GetUniqueProfiles(config.users) {
		allAvailableProfiles[profile] = true
	}

	var availableProfiles []string
	for profile := range allAvailableProfiles {
		availableProfiles = append(availableProfiles, profile)
	}
	slices.Sort(availableProfiles)

	tests := []struct {
		name            string
		activeProfiles  []string
		expectedInvalid []string
	}{
		{
			name:            "all_valid_profiles",
			activeProfiles:  []string{"work", "dev", "gaming"},
			expectedInvalid: []string{},
		},
		{
			name:            "mixed_valid_invalid",
			activeProfiles:  []string{"work", "invalid", "dev"},
			expectedInvalid: []string{"invalid"},
		},
		{
			name:            "all_keyword_always_valid",
			activeProfiles:  []string{"all", "invalid"},
			expectedInvalid: []string{"invalid"},
		},
		{
			name:            "completely_invalid",
			activeProfiles:  []string{"nonexistent1", "nonexistent2"},
			expectedInvalid: []string{"nonexistent1", "nonexistent2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid := ValidateProfiles(tt.activeProfiles, availableProfiles)

			// Handle nil vs empty slice comparison
			if len(invalid) == 0 && len(tt.expectedInvalid) == 0 {
				return
			}

			if !reflect.DeepEqual(invalid, tt.expectedInvalid) {
				t.Errorf("ValidateProfiles() = %v, expected %v", invalid, tt.expectedInvalid)
			}
		})
	}
}

// TestIntegration_ComplexProfileCombinations tests advanced profile usage patterns
func TestIntegration_ComplexProfileCombinations(t *testing.T) {
	// Create items with complex profile combinations
	complexPackages := []types.Package{
		{Name: "base-tool", Profiles: []string{}},                           // Base item
		{Name: "dev-tool", Profiles: []string{"dev"}},                       // Single profile
		{Name: "work-tool", Profiles: []string{"work"}},                     // Single profile
		{Name: "shared-tool", Profiles: []string{"dev", "work"}},            // Multi-profile
		{Name: "all-env-tool", Profiles: []string{"dev", "work", "gaming"}}, // Many profiles
	}

	tests := []struct {
		name             string
		activeProfiles   []string
		expectedPackages []string
	}{
		{
			name:             "dev_only",
			activeProfiles:   []string{"dev"},
			expectedPackages: []string{"base-tool", "dev-tool", "shared-tool", "all-env-tool"},
		},
		{
			name:             "work_only",
			activeProfiles:   []string{"work"},
			expectedPackages: []string{"base-tool", "work-tool", "shared-tool", "all-env-tool"},
		},
		{
			name:             "dev_and_work",
			activeProfiles:   []string{"dev", "work"},
			expectedPackages: []string{"base-tool", "dev-tool", "work-tool", "shared-tool", "all-env-tool"},
		},
		{
			name:             "gaming_only",
			activeProfiles:   []string{"gaming"},
			expectedPackages: []string{"base-tool", "all-env-tool"},
		},
		{
			name:             "no_profiles",
			activeProfiles:   []string{},
			expectedPackages: []string{"base-tool", "dev-tool", "work-tool", "shared-tool", "all-env-tool"},
		},
		{
			name:             "all_profiles",
			activeProfiles:   []string{"all"},
			expectedPackages: []string{"base-tool", "dev-tool", "work-tool", "shared-tool", "all-env-tool"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterByProfiles(complexPackages, tt.activeProfiles)
			packageNames := extractPackageNames(filtered)
			slices.Sort(packageNames)
			slices.Sort(tt.expectedPackages)

			if !reflect.DeepEqual(packageNames, tt.expectedPackages) {
				t.Errorf("Complex profiles %v: got %v, expected %v",
					tt.activeProfiles, packageNames, tt.expectedPackages)
			}
		})
	}
}

// Helper types and functions for integration tests

type workstationConfiguration struct {
	packages []types.Package
	services []types.Service
	files    []types.File
	users    []types.User
}

func createWorkstationConfiguration() workstationConfiguration {
	return workstationConfiguration{
		packages: []types.Package{
			// Base packages (no profiles) - always installed
			{Name: "vim", Profiles: []string{}},
			{Name: "git", Profiles: []string{}},
			{Name: "curl", Profiles: []string{}},

			// Work profile packages
			{Name: "docker", Profiles: []string{"work"}},
			{Name: "kubectl", Profiles: []string{"work"}},
			{Name: "terraform", Profiles: []string{"work"}},

			// Dev profile packages
			{Name: "nodejs", Profiles: []string{"dev"}},
			{Name: "python", Profiles: []string{"dev"}},
			{Name: "code", Profiles: []string{"dev"}},

			// Gaming profile packages
			{Name: "steam", Profiles: []string{"gaming"}},
			{Name: "discord", Profiles: []string{"gaming"}},

			// Multi-profile packages
			{Name: "tmux", Profiles: []string{"work", "dev"}},
		},
		services: []types.Service{
			// Base service
			{Name: "sshd", Profiles: []string{}},

			// Profile-specific services
			{Name: "docker", Profiles: []string{"work"}},
			{Name: "postgresql", Profiles: []string{"dev"}},
			{Name: "nginx", Profiles: []string{"work", "dev"}},
			{Name: "steam", Profiles: []string{"gaming"}},
		},
		files: []types.File{
			// Base file
			{Name: "bashrc", Profiles: []string{}},

			// Profile-specific files
			{Name: "work-ssh-config", Profiles: []string{"work"}},
			{Name: "dev-gitconfig", Profiles: []string{"dev"}},
		},
		users: []types.User{
			// Profile-specific users
			{Name: "developer", Profiles: []string{"dev"}},
		},
	}
}

func extractPackageNames(packages []types.Package) []string {
	var names []string
	for _, pkg := range packages {
		names = append(names, pkg.Name)
	}
	return names
}

func extractServiceNames(services []types.Service) []string {
	var names []string
	for _, svc := range services {
		names = append(names, svc.Name)
	}
	return names
}

// Benchmark integration scenarios
func BenchmarkIntegration_LargeConfiguration(b *testing.B) {
	// Create a large configuration similar to what enterprise users might have
	largeConfig := createLargeConfiguration()
	activeProfiles := []string{"work", "dev"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterByProfiles(largeConfig.packages, activeProfiles)
		FilterByProfiles(largeConfig.services, activeProfiles)
		FilterByProfiles(largeConfig.files, activeProfiles)
	}
}

func createLargeConfiguration() workstationConfiguration {
	config := workstationConfiguration{}

	// Create 1000 packages with various profile combinations
	profiles := []string{"work", "dev", "gaming", "personal", "server", "desktop"}

	for i := 0; i < 1000; i++ {
		pkg := types.Package{Name: "package-" + string(rune('A'+i%26)) + string(rune('0'+i/26))}

		// 20% base items (no profiles)
		if i%5 == 0 {
			pkg.Profiles = []string{}
		} else {
			// Assign 1-3 random profiles
			numProfiles := (i % 3) + 1
			for j := 0; j < numProfiles; j++ {
				profile := profiles[(i+j)%len(profiles)]
				if !slices.Contains(pkg.Profiles, profile) {
					pkg.Profiles = append(pkg.Profiles, profile)
				}
			}
		}
		config.packages = append(config.packages, pkg)
	}

	// Create 100 services
	for i := 0; i < 100; i++ {
		svc := types.Service{Name: "service-" + string(rune('A'+i%26))}
		if i%4 == 0 {
			svc.Profiles = []string{}
		} else {
			svc.Profiles = []string{profiles[i%len(profiles)]}
		}
		config.services = append(config.services, svc)
	}

	// Create 200 files
	for i := 0; i < 200; i++ {
		file := types.File{Name: "file-" + string(rune('A'+i%26))}
		if i%3 == 0 {
			file.Profiles = []string{}
		} else {
			file.Profiles = []string{profiles[i%len(profiles)]}
		}
		config.files = append(config.files, file)
	}

	return config
}
