package helpers

import (
	"reflect"
	"slices"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// Test fixtures
var (
	// Base items (no profiles) - always included
	basePackage = types.Package{Name: "vim", Profiles: []string{}}
	baseService = types.Service{Name: "sshd", Profiles: []string{}}
	baseFile    = types.File{Name: "bashrc", Profiles: []string{}}

	// Single profile items
	workPackage = types.Package{Name: "docker", Profiles: []string{"work"}}
	devService  = types.Service{Name: "postgresql", Profiles: []string{"dev"}}
	gameFile    = types.File{Name: "steam-config", Profiles: []string{"gaming"}}

	// Multi-profile items
	multiPackage = types.Package{Name: "tmux", Profiles: []string{"work", "dev"}}
	multiService = types.Service{Name: "nginx", Profiles: []string{"web", "server"}}

	// Various profile names to test flexibility
	personalPackage = types.Package{Name: "spotify", Profiles: []string{"personal"}}
	laptopService   = types.Service{Name: "bluetooth", Profiles: []string{"laptop"}}
	minimalFile     = types.File{Name: "minimal-vimrc", Profiles: []string{"minimal"}}
	// Package list items using Names field (more realistic)
	basePackageList  = types.Package{Names: []string{"vim", "git", "htop"}, Profiles: []string{}}
	workPackageList  = types.Package{Names: []string{"docker", "kubectl", "terraform"}, Profiles: []string{"work"}}
	gamePackageList  = types.Package{Names: []string{"steam", "discord"}, Profiles: []string{"gaming"}}
	multiPackageList = types.Package{Names: []string{"tmux", "screen"}, Profiles: []string{"work", "dev"}}
)

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name           string
		itemProfiles   []string
		activeProfiles []string
		expected       bool
		description    string
	}{
		// Base item scenarios (no profiles field)
		{
			name:           "base_item_no_active_profiles",
			itemProfiles:   []string{},
			activeProfiles: []string{},
			expected:       true,
			description:    "Base items (no profiles) should always be included when no profiles active",
		},
		{
			name:           "base_item_with_active_profiles",
			itemProfiles:   []string{},
			activeProfiles: []string{"work"},
			expected:       true,
			description:    "Base items (no profiles) should always be included even when profiles are active",
		},
		{
			name:           "base_item_with_multiple_active_profiles",
			itemProfiles:   []string{},
			activeProfiles: []string{"work", "dev", "gaming"},
			expected:       true,
			description:    "Base items (no profiles) should always be included with multiple active profiles",
		},

		// Single profile item scenarios
		{
			name:           "single_profile_no_active",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{},
			expected:       false,
			description:    "Profile items should be excluded when no profiles are active",
		},
		{
			name:           "single_profile_exact_match",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"work"},
			expected:       true,
			description:    "Profile items should be included when their profile matches active profile",
		},
		{
			name:           "single_profile_no_match",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"gaming"},
			expected:       false,
			description:    "Profile items should be excluded when their profile doesn't match active profiles",
		},
		{
			name:           "single_profile_partial_match_in_multiple",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"dev", "work", "gaming"},
			expected:       true,
			description:    "Profile items should be included when their profile matches one of multiple active profiles",
		},

		// Multi-profile item scenarios
		{
			name:           "multi_profile_no_active",
			itemProfiles:   []string{"work", "dev"},
			activeProfiles: []string{},
			expected:       false,
			description:    "Multi-profile items should be excluded when no profiles are active",
		},
		{
			name:           "multi_profile_one_match",
			itemProfiles:   []string{"work", "dev"},
			activeProfiles: []string{"work"},
			expected:       true,
			description:    "Multi-profile items should be included when one of their profiles matches active",
		},
		{
			name:           "multi_profile_multiple_matches",
			itemProfiles:   []string{"work", "dev"},
			activeProfiles: []string{"work", "dev"},
			expected:       true,
			description:    "Multi-profile items should be included when multiple profiles match active",
		},
		{
			name:           "multi_profile_no_match",
			itemProfiles:   []string{"work", "dev"},
			activeProfiles: []string{"gaming", "personal"},
			expected:       false,
			description:    "Multi-profile items should be excluded when none of their profiles match active",
		},

		// Special "all" keyword scenarios
		{
			name:           "all_keyword_with_base_item",
			itemProfiles:   []string{},
			activeProfiles: []string{"all"},
			expected:       true,
			description:    "Base items should be included when 'all' is active",
		},
		{
			name:           "all_keyword_with_profile_item",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"all"},
			expected:       true,
			description:    "Profile items should be included when 'all' is active",
		},
		{
			name:           "all_keyword_with_multi_profile_item",
			itemProfiles:   []string{"work", "dev", "gaming"},
			activeProfiles: []string{"all"},
			expected:       true,
			description:    "Multi-profile items should be included when 'all' is active",
		},
		{
			name:           "all_keyword_with_other_profiles",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"gaming", "all", "dev"},
			expected:       true,
			description:    "Any item should be included when 'all' is present with other profiles",
		},

		// Edge cases
		{
			name:           "empty_profile_in_item",
			itemProfiles:   []string{"", "work"},
			activeProfiles: []string{"work"},
			expected:       true,
			description:    "Items with empty string profiles should work when other profiles match",
		},
		{
			name:           "empty_profile_in_active",
			itemProfiles:   []string{"work"},
			activeProfiles: []string{"", "work"},
			expected:       true,
			description:    "Empty string in active profiles should not prevent matching",
		},
		{
			name:           "case_sensitive_profiles",
			itemProfiles:   []string{"Work"},
			activeProfiles: []string{"work"},
			expected:       false,
			description:    "Profile matching should be case sensitive",
		},

		// Real-world profile name scenarios
		{
			name:           "environment_based_profiles",
			itemProfiles:   []string{"desktop"},
			activeProfiles: []string{"laptop", "desktop"},
			expected:       true,
			description:    "Environment-based profile names should work",
		},
		{
			name:           "intensity_based_profiles",
			itemProfiles:   []string{"minimal"},
			activeProfiles: []string{"minimal"},
			expected:       true,
			description:    "Intensity-based profile names should work",
		},
		{
			name:           "context_based_profiles",
			itemProfiles:   []string{"home", "office"},
			activeProfiles: []string{"travel", "home"},
			expected:       true,
			description:    "Context-based profile names should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldInclude(tt.itemProfiles, tt.activeProfiles)
			if result != tt.expected {
				t.Errorf("ShouldInclude(%v, %v) = %v, expected %v\nDescription: %s",
					tt.itemProfiles, tt.activeProfiles, result, tt.expected, tt.description)
			}
		})
	}
}

func TestFilterByProfiles_Package(t *testing.T) {
	testPackages := []types.Package{
		basePackage,     // No profiles - always included
		workPackage,     // work profile
		personalPackage, // personal profile
		multiPackage,    // work, dev profiles
	}

	tests := []struct {
		name           string
		packages       []types.Package
		activeProfiles []string
		expectedNames  []string
		description    string
	}{
		{
			name:           "no_active_profiles",
			packages:       testPackages,
			activeProfiles: []string{},
			expectedNames:  []string{"vim"}, // Only base package
			description:    "Only base packages should be included when no profiles are active",
		},
		{
			name:           "work_profile_active",
			packages:       testPackages,
			activeProfiles: []string{"work"},
			expectedNames:  []string{"vim", "docker", "tmux"}, // base + work + multi
			description:    "Base packages and work-profile packages should be included",
		},
		{
			name:           "personal_profile_active",
			packages:       testPackages,
			activeProfiles: []string{"personal"},
			expectedNames:  []string{"vim", "spotify"}, // base + personal
			description:    "Base packages and personal-profile packages should be included",
		},
		{
			name:           "multiple_profiles_active",
			packages:       testPackages,
			activeProfiles: []string{"work", "personal"},
			expectedNames:  []string{"vim", "docker", "spotify", "tmux"}, // all packages
			description:    "Base packages and all matching profile packages should be included",
		},
		{
			name:           "all_keyword",
			packages:       testPackages,
			activeProfiles: []string{"all"},
			expectedNames:  []string{"vim", "docker", "spotify", "tmux"}, // everything
			description:    "All packages should be included when 'all' profile is active",
		},
		{
			name:           "non_existent_profile",
			packages:       testPackages,
			activeProfiles: []string{"nonexistent"},
			expectedNames:  []string{"vim"}, // Only base package
			description:    "Only base packages should be included when non-existent profile is active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByProfiles(tt.packages, tt.activeProfiles)

			// Extract names for comparison
			var resultNames []string
			for _, pkg := range result {
				resultNames = append(resultNames, pkg.Name)
			}

			// Sort both slices for reliable comparison
			slices.Sort(resultNames)
			slices.Sort(tt.expectedNames)

			if !reflect.DeepEqual(resultNames, tt.expectedNames) {
				t.Errorf("FilterByProfiles() = %v, expected %v\nDescription: %s",
					resultNames, tt.expectedNames, tt.description)
			}
		})
	}
}

func TestFilterByProfiles_Service(t *testing.T) {
	testServices := []types.Service{
		baseService,  // No profiles
		devService,   // dev profile
		multiService, // web, server profiles
	}

	tests := []struct {
		name           string
		services       []types.Service
		activeProfiles []string
		expectedNames  []string
	}{
		{
			name:           "dev_profile_active",
			services:       testServices,
			activeProfiles: []string{"dev"},
			expectedNames:  []string{"sshd", "postgresql"},
		},
		{
			name:           "server_profile_active",
			services:       testServices,
			activeProfiles: []string{"server"},
			expectedNames:  []string{"sshd", "nginx"},
		},
		{
			name:           "all_profiles",
			services:       testServices,
			activeProfiles: []string{"all"},
			expectedNames:  []string{"sshd", "postgresql", "nginx"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByProfiles(tt.services, tt.activeProfiles)

			var resultNames []string
			for _, svc := range result {
				resultNames = append(resultNames, svc.Name)
			}

			slices.Sort(resultNames)
			slices.Sort(tt.expectedNames)

			if !reflect.DeepEqual(resultNames, tt.expectedNames) {
				t.Errorf("FilterByProfiles() = %v, expected %v", resultNames, tt.expectedNames)
			}
		})
	}
}

func TestFilterByProfiles_PackageLists(t *testing.T) {
	testPackageLists := []types.Package{
		basePackageList,  // Names: ["vim", "git", "htop"], Profiles: []
		workPackageList,  // Names: ["docker", "kubectl", "terraform"], Profiles: ["work"]
		gamePackageList,  // Names: ["steam", "discord"], Profiles: ["gaming"]
		multiPackageList, // Names: ["tmux", "screen"], Profiles: ["work", "dev"]
	}

	tests := []struct {
		name           string
		packages       []types.Package
		activeProfiles []string
		expectedNames  [][]string // Expected Names field for each returned package
		description    string
	}{
		{
			name:           "no_active_profiles_lists",
			packages:       testPackageLists,
			activeProfiles: []string{},
			expectedNames:  [][]string{{"vim", "git", "htop"}}, // Only base package list
			description:    "Only base package lists should be included when no profiles are active",
		},
		{
			name:           "work_profile_active_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"work"},
			expectedNames: [][]string{
				{"vim", "git", "htop"},             // base
				{"docker", "kubectl", "terraform"}, // work
				{"tmux", "screen"},                 // multi (work, dev)
			},
			description: "Base and work-profile package lists should be included",
		},
		{
			name:           "gaming_profile_active_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"gaming"},
			expectedNames: [][]string{
				{"vim", "git", "htop"}, // base
				{"steam", "discord"},   // gaming
			},
			description: "Base and gaming-profile package lists should be included",
		},
		{
			name:           "dev_profile_active_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"dev"},
			expectedNames: [][]string{
				{"vim", "git", "htop"}, // base
				{"tmux", "screen"},     // multi (work, dev)
			},
			description: "Base and dev-profile package lists should be included",
		},
		{
			name:           "multiple_profiles_active_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"work", "gaming"},
			expectedNames: [][]string{
				{"vim", "git", "htop"},             // base
				{"docker", "kubectl", "terraform"}, // work
				{"steam", "discord"},               // gaming
				{"tmux", "screen"},                 // multi (work, dev)
			},
			description: "Base and all matching profile package lists should be included",
		},
		{
			name:           "all_keyword_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"all"},
			expectedNames: [][]string{
				{"vim", "git", "htop"},             // base
				{"docker", "kubectl", "terraform"}, // work
				{"steam", "discord"},               // gaming
				{"tmux", "screen"},                 // multi
			},
			description: "All package lists should be included when 'all' profile is active",
		},
		{
			name:           "non_existent_profile_lists",
			packages:       testPackageLists,
			activeProfiles: []string{"nonexistent"},
			expectedNames:  [][]string{{"vim", "git", "htop"}}, // Only base
			description:    "Only base package lists should be included when non-existent profile is active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByProfiles(tt.packages, tt.activeProfiles)

			if len(result) != len(tt.expectedNames) {
				t.Errorf("FilterByProfiles() returned %d package lists, expected %d\nDescription: %s",
					len(result), len(tt.expectedNames), tt.description)
				return
			}

			// Verify each package list's Names field
			for i, pkg := range result {
				if !reflect.DeepEqual(pkg.Names, tt.expectedNames[i]) {
					t.Errorf("FilterByProfiles() package list %d Names = %v, expected %v\nDescription: %s",
						i, pkg.Names, tt.expectedNames[i], tt.description)
				}
			}
		})
	}
}

func TestFilterByProfiles_EmptyInput(t *testing.T) {
	// Test with empty package slice
	result := FilterByProfiles([]types.Package{}, []string{"work"})
	if len(result) != 0 {
		t.Errorf("FilterByProfiles([], [work]) should return empty slice, got %v", result)
	}

	// Test with nil package slice (should not panic)
	result = FilterByProfiles([]types.Package(nil), []string{"work"})
	if len(result) != 0 {
		t.Errorf("FilterByProfiles(nil, [work]) should return empty slice, got %v", result)
	}
}

func TestGetUniqueProfiles(t *testing.T) {
	tests := []struct {
		name     string
		packages []types.Package
		expected []string
	}{
		{
			name:     "mixed_profiles",
			packages: []types.Package{basePackage, workPackage, personalPackage, multiPackage},
			expected: []string{"dev", "personal", "work"}, // sorted unique profiles
		},
		{
			name: "duplicate_profiles",
			packages: []types.Package{
				{Name: "pkg1", Profiles: []string{"work", "dev"}},
				{Name: "pkg2", Profiles: []string{"dev", "work"}},
				{Name: "pkg3", Profiles: []string{"work"}},
			},
			expected: []string{"dev", "work"},
		},
		{
			name:     "empty_profiles",
			packages: []types.Package{basePackage, {Name: "pkg1", Profiles: []string{}}},
			expected: []string{},
		},
		{
			name: "profiles_with_empty_strings",
			packages: []types.Package{
				{Name: "pkg1", Profiles: []string{"", "work", ""}},
				{Name: "pkg2", Profiles: []string{"dev", ""}},
			},
			expected: []string{"dev", "work"}, // empty strings filtered out
		},
		{
			name:     "no_packages",
			packages: []types.Package{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUniqueProfiles(tt.packages)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both are effectively empty
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetUniqueProfiles() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetUniqueProfiles_PackageLists(t *testing.T) {
	tests := []struct {
		name     string
		packages []types.Package
		expected []string
	}{
		{
			name:     "package_lists_with_profiles",
			packages: []types.Package{basePackageList, workPackageList, gamePackageList, multiPackageList},
			expected: []string{"dev", "gaming", "work"}, // sorted unique profiles from lists
		},
		{
			name: "mixed_single_and_lists",
			packages: []types.Package{
				basePackage,     // single package with Name field
				basePackageList, // package list with Names field
				workPackage,     // single package
				workPackageList, // package list
			},
			expected: []string{"work"}, // unique profiles from both single and list packages
		},
		{
			name: "only_package_lists_no_profiles",
			packages: []types.Package{
				{Names: []string{"pkg1", "pkg2"}, Profiles: []string{}},
				{Names: []string{"pkg3", "pkg4"}, Profiles: []string{}},
			},
			expected: []string{}, // no profiles defined
		},
		{
			name: "package_lists_duplicate_profiles",
			packages: []types.Package{
				{Names: []string{"tools1"}, Profiles: []string{"work", "dev"}},
				{Names: []string{"tools2"}, Profiles: []string{"dev", "work"}},
				{Names: []string{"tools3"}, Profiles: []string{"work"}},
			},
			expected: []string{"dev", "work"}, // deduplicated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUniqueProfiles(tt.packages)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both are effectively empty
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetUniqueProfiles() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetUniqueProfiles_MultipleTypes(t *testing.T) {
	// Test with different types to ensure generic function works
	testPackages := []types.Package{
		{Name: "pkg1", Profiles: []string{"work"}},
		{Name: "pkg2", Profiles: []string{"dev"}},
	}

	testServices := []types.Service{
		{Name: "svc1", Profiles: []string{"server"}},
		{Name: "svc2", Profiles: []string{"work"}}, // duplicate with package
	}

	testFiles := []types.File{
		{Name: "file1", Profiles: []string{"personal"}},
	}

	// Test each type separately
	pkgProfiles := GetUniqueProfiles(testPackages)
	svcProfiles := GetUniqueProfiles(testServices)
	fileProfiles := GetUniqueProfiles(testFiles)

	expectedPkg := []string{"dev", "work"}
	expectedSvc := []string{"server", "work"}
	expectedFile := []string{"personal"}

	if !reflect.DeepEqual(pkgProfiles, expectedPkg) {
		t.Errorf("Package profiles = %v, expected %v", pkgProfiles, expectedPkg)
	}

	if !reflect.DeepEqual(svcProfiles, expectedSvc) {
		t.Errorf("Service profiles = %v, expected %v", svcProfiles, expectedSvc)
	}

	if !reflect.DeepEqual(fileProfiles, expectedFile) {
		t.Errorf("File profiles = %v, expected %v", fileProfiles, expectedFile)
	}
}

func TestValidateProfiles(t *testing.T) {
	availableProfiles := []string{"work", "dev", "gaming", "personal"}

	tests := []struct {
		name              string
		activeProfiles    []string
		availableProfiles []string
		expectedInvalid   []string
	}{
		{
			name:              "all_valid_profiles",
			activeProfiles:    []string{"work", "dev"},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{},
		},
		{
			name:              "some_invalid_profiles",
			activeProfiles:    []string{"work", "invalid", "dev", "nonexistent"},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{"invalid", "nonexistent"},
		},
		{
			name:              "all_invalid_profiles",
			activeProfiles:    []string{"invalid1", "invalid2"},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{"invalid1", "invalid2"},
		},
		{
			name:              "all_keyword_is_valid",
			activeProfiles:    []string{"all", "invalid"},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{"invalid"}, // "all" should not be in invalid list
		},
		{
			name:              "empty_active_profiles",
			activeProfiles:    []string{},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{},
		},
		{
			name:              "empty_available_profiles",
			activeProfiles:    []string{"work"},
			availableProfiles: []string{},
			expectedInvalid:   []string{"work"},
		},
		{
			name:              "only_all_keyword",
			activeProfiles:    []string{"all"},
			availableProfiles: availableProfiles,
			expectedInvalid:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateProfiles(tt.activeProfiles, tt.availableProfiles)

			// Handle nil vs empty slice comparison
			if len(result) == 0 && len(tt.expectedInvalid) == 0 {
				return // Both are effectively empty
			}

			if !reflect.DeepEqual(result, tt.expectedInvalid) {
				t.Errorf("ValidateProfiles() = %v, expected %v", result, tt.expectedInvalid)
			}
		})
	}
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("unicode_profile_names", func(t *testing.T) {
		unicodePackage := types.Package{
			Name:     "unicode-test",
			Profiles: []string{"工作", "домой", "🏠"},
		}

		result := ShouldInclude(unicodePackage.Profiles, []string{"工作"})
		if !result {
			t.Error("Unicode profile names should be supported")
		}

		profiles := GetUniqueProfiles([]types.Package{unicodePackage})
		// Note: The actual sort order may vary for Unicode, so we just check that all profiles are present
		if len(profiles) != 3 {
			t.Errorf("Expected 3 unique unicode profiles, got %d: %v", len(profiles), profiles)
		}

		// Verify all expected profiles are present
		expectedProfiles := []string{"工作", "домой", "🏠"}
		for _, expected := range expectedProfiles {
			found := false
			for _, profile := range profiles {
				if profile == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected profile %s not found in result %v", expected, profiles)
			}
		}
	})

	t.Run("very_long_profile_names", func(t *testing.T) {
		longProfileName := "this-is-a-very-long-profile-name-that-might-be-used-in-some-edge-cases-where-users-want-to-be-very-descriptive-about-their-profile-names"
		longPackage := types.Package{
			Name:     "long-profile-test",
			Profiles: []string{longProfileName},
		}

		result := ShouldInclude(longPackage.Profiles, []string{longProfileName})
		if !result {
			t.Error("Very long profile names should be supported")
		}
	})

	t.Run("many_profiles_on_single_item", func(t *testing.T) {
		manyProfiles := make([]string, 100)
		for i := 0; i < 100; i++ {
			manyProfiles[i] = "profile" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		}

		manyProfilePackage := types.Package{
			Name:     "many-profiles-test",
			Profiles: manyProfiles,
		}

		result := ShouldInclude(manyProfilePackage.Profiles, []string{"profileA0"})
		if !result {
			t.Error("Items with many profiles should still work correctly")
		}

		// Test that all profiles are found
		foundProfiles := GetUniqueProfiles([]types.Package{manyProfilePackage})
		if len(foundProfiles) != 100 {
			t.Errorf("Expected 100 unique profiles, got %d", len(foundProfiles))
		}
	})
}

// Benchmark tests
func BenchmarkShouldInclude(b *testing.B) {
	itemProfiles := []string{"work", "dev", "testing"}
	activeProfiles := []string{"personal", "work", "gaming"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldInclude(itemProfiles, activeProfiles)
	}
}

func BenchmarkFilterByProfiles(b *testing.B) {
	// Create a large set of test packages
	packages := make([]types.Package, 1000)
	profiles := []string{"work", "dev", "gaming", "personal", "server", "desktop", "minimal", "full"}

	for i := 0; i < 1000; i++ {
		packages[i] = types.Package{
			Name:     "package" + string(rune(i)),
			Profiles: []string{profiles[i%len(profiles)]},
		}
	}

	activeProfiles := []string{"work", "dev"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterByProfiles(packages, activeProfiles)
	}
}

func BenchmarkGetUniqueProfiles(b *testing.B) {
	// Create packages with overlapping profiles
	packages := make([]types.Package, 500)
	profiles := []string{"work", "dev", "gaming", "personal"}

	for i := 0; i < 500; i++ {
		packages[i] = types.Package{
			Name:     "package" + string(rune(i)),
			Profiles: []string{profiles[i%len(profiles)], profiles[(i+1)%len(profiles)]},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetUniqueProfiles(packages)
	}
}
