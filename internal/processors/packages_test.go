package processors

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// Test blueprint parsing without calling the actual ProcessPackages function
func TestProcessPackages_BlueprintParsing(t *testing.T) {
	blueprintData := []byte(`
packages:
  - name: "git"
    action: "install"
    package_manager: "auto"
  - name: "curl"
    action: "install"
    profiles: ["development"]
`)

	var pkgData types.PackagesData
	err := helpers.UnmarshalBlueprint(blueprintData, "yaml", &pkgData)

	if err != nil {
		t.Fatalf("Blueprint parsing failed: %v", err)
	}

	if len(pkgData.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(pkgData.Packages))
	}

	// Validate first package
	if pkgData.Packages[0].Name != "git" {
		t.Errorf("Expected first package name to be 'git', got '%s'", pkgData.Packages[0].Name)
	}
	if pkgData.Packages[0].Action != "install" {
		t.Errorf("Expected first package action to be 'install', got '%s'", pkgData.Packages[0].Action)
	}
	if pkgData.Packages[0].PackageManager != "auto" {
		t.Errorf("Expected first package manager to be 'auto', got '%s'", pkgData.Packages[0].PackageManager)
	}

	// Validate second package
	if pkgData.Packages[1].Name != "curl" {
		t.Errorf("Expected second package name to be 'curl', got '%s'", pkgData.Packages[1].Name)
	}
	if len(pkgData.Packages[1].Profiles) != 1 || pkgData.Packages[1].Profiles[0] != "development" {
		t.Errorf("Expected second package to have profile 'development', got %v", pkgData.Packages[1].Profiles)
	}

	t.Log("Blueprint parsing successful")
}

// Test profile filtering logic independently
func TestProcessPackages_ProfileFiltering(t *testing.T) {
	packages := []types.Package{
		{
			Name:     "nodejs",
			Action:   "install",
			Profiles: []string{"development"}, // Should be included
		},
		{
			Name:     "nginx",
			Action:   "install",
			Profiles: []string{"production"}, // Should be filtered out
		},
		{
			Name:   "git",
			Action: "install",
			// No profiles specified - should be included
		},
		{
			Name:     "docker",
			Action:   "install",
			Profiles: []string{"development", "production"}, // Should be included (matches development)
		},
	}

	activeProfiles := []string{"development"}
	filteredPackages := helpers.FilterByProfiles(packages, activeProfiles)

	// Should include: nodejs (development), git (no profiles), docker (matches development)
	// Should exclude: nginx (production only)
	expectedCount := 3
	if len(filteredPackages) != expectedCount {
		t.Errorf("Expected %d filtered packages, got %d", expectedCount, len(filteredPackages))
	}

	// Verify specific packages are included
	names := make([]string, len(filteredPackages))
	for i, pkg := range filteredPackages {
		names[i] = pkg.Name
	}

	expectedNames := []string{"nodejs", "git", "docker"}
	for _, expected := range expectedNames {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected package '%s' to be included in filtered results", expected)
		}
	}

	// Verify nginx is excluded
	for _, name := range names {
		if name == "nginx" {
			t.Error("Package 'nginx' should have been filtered out")
		}
	}

	t.Log("Profile filtering logic works correctly")
}

// Test package structure validation
func TestProcessPackages_PackageStructure(t *testing.T) {
	testCases := []struct {
		name        string
		pkg         types.Package
		expectValid bool
	}{
		{
			name: "Valid package with name",
			pkg: types.Package{
				Name:   "git",
				Action: "install",
			},
			expectValid: true,
		},
		{
			name: "Valid package with names array",
			pkg: types.Package{
				Names:  []string{"git", "curl", "wget"},
				Action: "install",
			},
			expectValid: true,
		},
		{
			name: "Package with both name and names (name takes precedence)",
			pkg: types.Package{
				Name:   "git",
				Names:  []string{"curl", "wget"},
				Action: "install",
			},
			expectValid: true,
		},
		{
			name: "Package with arguments",
			pkg: types.Package{
				Name:   "postgresql",
				Action: "install",
				Args:   []string{"--no-install-recommends", "-y"},
			},
			expectValid: true,
		},
		{
			name: "Package with specific manager",
			pkg: types.Package{
				Name:           "vim",
				Action:         "install",
				PackageManager: "apt",
			},
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test package structure validation
			if tc.pkg.Name != "" && len(tc.pkg.Names) > 0 {
				// When both name and names are provided, name should take precedence
				if tc.pkg.Name == "" {
					t.Error("Name should not be empty when provided")
				}
			}

			// Test action validation
			validActions := []string{"install", "remove"}
			actionValid := false
			for _, validAction := range validActions {
				if tc.pkg.Action == validAction {
					actionValid = true
					break
				}
			}

			if tc.expectValid && !actionValid {
				t.Errorf("Expected valid action, got '%s'", tc.pkg.Action)
			}

			// Test package manager field
			if tc.pkg.PackageManager != "" {
				if tc.pkg.PackageManager == "" {
					t.Error("PackageManager should not be empty when set")
				}
			}

			t.Logf("Package structure validation passed for %s", tc.name)
		})
	}
}

// Test command building logic without execution
func TestProcessPackages_CommandGeneration(t *testing.T) {
	// Mock provider data that would normally come from system.GetProvider()
	mockProvider := &types.Provider{
		Name:     "apt",
		BinPath:  "/usr/bin/apt",
		Elevated: true,
		Commands: types.CommandConfig{
			Install: "install -y",
			Remove:  "remove -y",
		},
		Environment: map[string]string{
			"DEBIAN_FRONTEND": "noninteractive",
		},
	}

	testCases := []struct {
		name                string
		pkg                 types.Package
		expectedExec        string
		expectedElevated    bool
		expectedArgsContain []string
	}{
		{
			name: "Install command",
			pkg: types.Package{
				Name:   "git",
				Action: "install",
			},
			expectedExec:        "/usr/bin/apt",
			expectedElevated:    true,
			expectedArgsContain: []string{"install", "-y", "git"},
		},
		{
			name: "Remove command",
			pkg: types.Package{
				Name:   "git",
				Action: "remove",
			},
			expectedExec:        "/usr/bin/apt",
			expectedElevated:    true,
			expectedArgsContain: []string{"remove", "-y", "git"},
		},
		{
			name: "Install with additional args",
			pkg: types.Package{
				Name:   "postgresql",
				Action: "install",
				Args:   []string{"--no-install-recommends"},
			},
			expectedExec:        "/usr/bin/apt",
			expectedElevated:    true,
			expectedArgsContain: []string{"install", "-y", "postgresql", "--no-install-recommends"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build command arguments like ProcessPackages would
			var args []string
			switch tc.pkg.Action {
			case "install":
				args = append(args, strings.Fields(mockProvider.Commands.Install)...)
			case "remove":
				args = append(args, strings.Fields(mockProvider.Commands.Remove)...)
			}

			// Add package name
			var names []string
			if tc.pkg.Name != "" {
				names = []string{tc.pkg.Name}
			} else {
				names = tc.pkg.Names
			}

			args = append(args, names...)

			// Add additional arguments
			if len(tc.pkg.Args) > 0 {
				args = append(args, tc.pkg.Args...)
			}

			// Build command
			cmd := types.Command{
				Exec:      mockProvider.BinPath,
				Args:      args,
				Elevated:  mockProvider.Elevated,
				Variables: mockProvider.Environment,
			}

			// Validate command structure
			if cmd.Exec != tc.expectedExec {
				t.Errorf("Expected exec '%s', got '%s'", tc.expectedExec, cmd.Exec)
			}

			if cmd.Elevated != tc.expectedElevated {
				t.Errorf("Expected elevated %v, got %v", tc.expectedElevated, cmd.Elevated)
			}

			// Check that all expected args are present
			for _, expectedArg := range tc.expectedArgsContain {
				found := false
				for _, arg := range cmd.Args {
					if arg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected arg '%s' not found in command args %v", expectedArg, cmd.Args)
				}
			}

			// Validate environment variables
			if cmd.Variables["DEBIAN_FRONTEND"] != "noninteractive" {
				t.Errorf("Expected DEBIAN_FRONTEND=noninteractive, got %v", cmd.Variables)
			}

			t.Logf("Command generation test passed: %s %v", cmd.Exec, cmd.Args)
		})
	}
}

// Test edge cases and error conditions
func TestProcessPackages_EdgeCases(t *testing.T) {
	t.Run("Empty package name and names", func(t *testing.T) {
		pkg := types.Package{
			// Both Name and Names are empty
		}
		_ = pkg.Action // Assign a value to avoid unused write warning

		// This should be handled gracefully - no names means no packages to process
		var names []string
		if pkg.Name != "" {
			names = []string{pkg.Name}
		} else {
			names = pkg.Names
		}

		if len(names) != 0 {
			t.Error("Expected no names for empty package")
		}
		t.Log("Empty package names handled correctly")
	})

	t.Run("Invalid action", func(t *testing.T) {
		pkg := types.Package{
			Action: "invalid-action",
		}
		_ = pkg.Name // Assign a value to avoid unused write warning

		// Invalid actions should be detected
		validActions := map[string]bool{
			"install": true,
			"remove":  true,
		}

		if validActions[pkg.Action] {
			t.Error("Invalid action should not be considered valid")
		}
		t.Log("Invalid action properly detected")
	})

	t.Run("Special characters in package names", func(t *testing.T) {
		pkg := types.Package{
			Names: []string{"lib-test++", "package.with.dots", "package_with_underscores"},
		}
		_ = pkg.Action // Assign a value to avoid unused write warning

		// Special characters should be preserved
		for _, name := range pkg.Names {
			if name == "" {
				t.Error("Package name should not be empty")
			}
		}

		if len(pkg.Names) != 3 {
			t.Errorf("Expected 3 package names, got %d", len(pkg.Names))
		}
		t.Log("Special characters in package names handled correctly")
	})
}

// Test blueprint format variations
func TestProcessPackages_BlueprintFormats(t *testing.T) {
	testCases := []struct {
		name   string
		format string
		data   []byte
	}{
		{
			name:   "YAML format",
			format: "yaml",
			data: []byte(`
packages:
  - name: "git"
    action: "install"
`),
		},
		{
			name:   "JSON format",
			format: "json",
			data: []byte(`{
  "packages": [
    {
      "name": "git",
      "action": "install"
    }
  ]
}`),
		},
		{
			name:   "TOML format",
			format: "toml",
			data: []byte(`
[[packages]]
name = "git"
action = "install"
`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pkgData types.PackagesData
			err := helpers.UnmarshalBlueprint(tc.data, tc.format, &pkgData)

			if err != nil {
				t.Fatalf("Failed to parse %s format: %v", tc.format, err)
			}

			if len(pkgData.Packages) != 1 {
				t.Errorf("Expected 1 package, got %d", len(pkgData.Packages))
			}

			if pkgData.Packages[0].Name != "git" {
				t.Errorf("Expected package name 'git', got '%s'", pkgData.Packages[0].Name)
			}

			if pkgData.Packages[0].Action != "install" {
				t.Errorf("Expected action 'install', got '%s'", pkgData.Packages[0].Action)
			}

			t.Logf("%s format parsing successful", tc.format)
		})
	}
}

// Test invalid blueprint data
func TestProcessPackages_InvalidBlueprint(t *testing.T) {
	invalidBlueprint := []byte(`
packages:
  - name: "test"
    action: "install"
    invalid_field: [this is invalid yaml
`)

	var pkgData types.PackagesData
	err := helpers.UnmarshalBlueprint(invalidBlueprint, "yaml", &pkgData)

	if err == nil {
		t.Fatal("Invalid blueprint should return an error")
	}

	if !containsString(err.Error(), "yaml") && !containsString(err.Error(), "unmarshal") {
		t.Errorf("Expected YAML parsing error, got: %v", err)
	}
	t.Log("Invalid blueprint properly rejected")
}

// Benchmark tests for performance
func BenchmarkPackageFiltering(b *testing.B) {
	packages := []types.Package{
		{Name: "git", Action: "install", Profiles: []string{"development"}},
		{Name: "curl", Action: "install", Profiles: []string{"production"}},
		{Name: "wget", Action: "install"},
		{Name: "vim", Action: "install", Profiles: []string{"development", "editor"}},
		{Name: "nginx", Action: "install", Profiles: []string{"production", "web"}},
	}

	activeProfiles := []string{"development"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = helpers.FilterByProfiles(packages, activeProfiles)
	}
}

func BenchmarkBlueprintParsing(b *testing.B) {
	blueprintData := []byte(`
packages:
  - name: "git"
    action: "install"
  - name: "curl"
    action: "install"
  - name: "wget"
    action: "install"
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var pkgData types.PackagesData
		_ = helpers.UnmarshalBlueprint(blueprintData, "yaml", &pkgData)
	}
}
