package system

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/types"
)

func TestLoadEmbeddedProviders(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	// Verify we loaded some providers
	if len(providers) == 0 {
		t.Fatal("LoadEmbeddedProviders() returned no providers")
	}

	t.Logf("Loaded %d providers", len(providers))

	// Test that we have expected core providers
	expectedProviders := []string{"pacman", "yay", "paru", "aura", "trizen", "pamac"}
	for _, expected := range expectedProviders {
		provider, exists := providers[expected]
		if !exists {
			t.Errorf("Expected provider %s not found", expected)
			continue
		}

		// Validate provider structure
		if provider.Name != expected {
			t.Errorf("Provider %s has incorrect name: got %s, want %s", expected, provider.Name, expected)
		}

		// Validate that required fields are set
		if provider.Detection.Binary == "" {
			t.Errorf("Provider %s missing detection binary", expected)
		}

		// Validate command structure
		if provider.Commands.Install == "" {
			t.Errorf("Provider %s missing install command", expected)
		}
		if provider.Commands.Update == "" {
			t.Errorf("Provider %s missing update command", expected)
		}
		if provider.Commands.Remove == "" {
			t.Errorf("Provider %s missing remove command", expected)
		}
	}
}

func TestLoadEmbeddedProviders_ProviderStructure(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	// Test a specific provider (pacman) for detailed structure validation
	pacman, exists := providers["pacman"]
	if !exists {
		t.Fatal("pacman provider not found")
	}

	// Test detection configuration
	if pacman.Detection.Binary != "pacman" {
		t.Errorf("pacman detection binary: got %s, want pacman", pacman.Detection.Binary)
	}

	expectedFiles := []string{"/etc/pacman.conf", "/var/lib/pacman"}
	if len(pacman.Detection.Files) != len(expectedFiles) {
		t.Errorf("pacman detection files count: got %d, want %d", len(pacman.Detection.Files), len(expectedFiles))
	}

	expectedDistros := []string{"arch", "cachyos", "linux/cachyos", "manjaro"}
	if len(pacman.Detection.Distributions) != len(expectedDistros) {
		t.Errorf("pacman detection distributions count: got %d, want %d", len(pacman.Detection.Distributions), len(expectedDistros))
	}

	// Test commands
	expectedCommands := map[string]string{
		"install": "-Sy --noconfirm",
		"update":  "-Syu --noconfirm",
		"remove":  "-R --noconfirm",
		"list":    "-Q",
		"search":  "-Ss",
		"clean":   "-Sc --noconfirm",
	}

	if pacman.Commands.Install != expectedCommands["install"] {
		t.Errorf("pacman install command: got %s, want %s", pacman.Commands.Install, expectedCommands["install"])
	}
	if pacman.Commands.Update != expectedCommands["update"] {
		t.Errorf("pacman update command: got %s, want %s", pacman.Commands.Update, expectedCommands["update"])
	}
	if pacman.Commands.Remove != expectedCommands["remove"] {
		t.Errorf("pacman remove command: got %s, want %s", pacman.Commands.Remove, expectedCommands["remove"])
	}
	if pacman.Commands.List != expectedCommands["list"] {
		t.Errorf("pacman list command: got %s, want %s", pacman.Commands.List, expectedCommands["list"])
	}
	if pacman.Commands.Search != expectedCommands["search"] {
		t.Errorf("pacman search command: got %s, want %s", pacman.Commands.Search, expectedCommands["search"])
	}
	if pacman.Commands.Clean != expectedCommands["clean"] {
		t.Errorf("pacman clean command: got %s, want %s", pacman.Commands.Clean, expectedCommands["clean"])
	}

	// Test elevated privilege requirement
	if !pacman.Elevated {
		t.Error("pacman should require elevated privileges")
	}

	// Test core packages
	if len(pacman.CorePackages) == 0 {
		t.Error("pacman should have core packages defined")
	}

	// Check for specific core packages
	if _, exists := pacman.CorePackages["openssl"]; !exists {
		t.Error("pacman should have openssl core package defined")
	}
	if _, exists := pacman.CorePackages["build-essentials"]; !exists {
		t.Error("pacman should have build-essentials core package defined")
	}

	// Test repository configuration
	if pacman.Repository.Paths.Sources != "/etc/pacman.d" {
		t.Errorf("pacman repository sources path: got %s, want /etc/pacman.d", pacman.Repository.Paths.Sources)
	}
	if pacman.Repository.Paths.Keys != "/etc/pacman.d/gnupg" {
		t.Errorf("pacman repository keys path: got %s, want /etc/pacman.d/gnupg", pacman.Repository.Paths.Keys)
	}

	// Test repository add steps
	if len(pacman.Repository.Add.Steps) == 0 {
		t.Error("pacman should have repository add steps defined")
	}

	// Test repository remove steps
	if len(pacman.Repository.Remove.Steps) == 0 {
		t.Error("pacman should have repository remove steps defined")
	}
}

func TestLoadEmbeddedProviders_AllProvidersValid(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	for name, provider := range providers {
		t.Run(name, func(t *testing.T) {
			// Test provider name consistency
			if provider.Name != name {
				t.Errorf("Provider key %s doesn't match provider name %s", name, provider.Name)
			}

			// Test required fields
			if provider.Detection.Binary == "" {
				t.Error("Provider missing detection binary")
			}

			// Test at least basic commands are present
			if provider.Commands.Install == "" {
				t.Error("Provider missing install command")
			}
			if provider.Commands.Update == "" {
				t.Error("Provider missing update command")
			}
			if provider.Commands.Remove == "" {
				t.Error("Provider missing remove command")
			}

			// Test that detection has some criteria
			hasDetectionCriteria := provider.Detection.Binary != "" ||
				len(provider.Detection.Files) > 0 ||
				len(provider.Detection.Distributions) > 0

			if !hasDetectionCriteria {
				t.Error("Provider should have at least one detection criterion")
			}

			// Test install/remove steps if present
			if len(provider.Install.Steps) > 0 {
				for i, step := range provider.Install.Steps {
					if step.Action == "" {
						t.Errorf("Install step %d missing action", i)
					}
					if step.Action == "command" && step.Exec == "" {
						t.Errorf("Install step %d has command action but missing exec", i)
					}
				}
			}

			if len(provider.Remove.Steps) > 0 {
				for i, step := range provider.Remove.Steps {
					if step.Action == "" {
						t.Errorf("Remove step %d missing action", i)
					}
					if step.Action == "command" && step.Exec == "" {
						t.Errorf("Remove step %d has command action but missing exec", i)
					}
				}
			}

			// Test repository steps if present
			if len(provider.Repository.Add.Steps) > 0 {
				for i, step := range provider.Repository.Add.Steps {
					if step.Action == "" {
						t.Errorf("Repository add step %d missing action", i)
					}
					// Validate action types according to docs
					validActions := []string{"download", "write", "append", "command", "remove",
						"remove_line", "remove_section", "mkdir", "chmod", "chown", "symlink", "copy"}
					isValidAction := false
					for _, validAction := range validActions {
						if step.Action == validAction {
							isValidAction = true
							break
						}
					}
					if !isValidAction {
						t.Errorf("Repository add step %d has invalid action: %s", i, step.Action)
					}
				}
			}

			if len(provider.Repository.Remove.Steps) > 0 {
				for i, step := range provider.Repository.Remove.Steps {
					if step.Action == "" {
						t.Errorf("Repository remove step %d missing action", i)
					}
				}
			}

			// Test core packages structure
			if len(provider.CorePackages) > 0 {
				// Check for expected core package categories, but allow empty arrays
				// Some providers might have empty categories for certain package types
				expectedCategories := []string{"openssl", "build-essentials"}
				for _, category := range expectedCategories {
					if packages, exists := provider.CorePackages[category]; exists {
						// Log info about empty categories but don't fail the test
						// Some providers like cargo might not need traditional system packages
						if len(packages) == 0 {
							t.Logf("Core package category %s exists but is empty for provider %s", category, name)
						}
					}
				}
			}

			// Test alternatives structure if present
			for distro, alternatives := range provider.Alternatives {
				if distro == "" {
					t.Error("Alternative distribution name cannot be empty")
				}
				if len(alternatives.CorePackages) == 0 {
					t.Errorf("Alternative for distribution %s has no core packages", distro)
				}
			}
		})
	}
}

func TestGetEmbeddedProviderFiles(t *testing.T) {
	files, err := GetEmbeddedProviderFiles()
	if err != nil {
		t.Fatalf("GetEmbeddedProviderFiles() failed: %v", err)
	}

	// Verify we got some files
	if len(files) == 0 {
		t.Fatal("GetEmbeddedProviderFiles() returned no files")
	}

	t.Logf("Found %d embedded provider files", len(files))

	// Verify expected files exist
	expectedFiles := []string{"pacman.toml", "yay.toml", "paru.toml", "aura.toml", "trizen.toml", "pamac.toml"}
	for _, expected := range expectedFiles {
		content, exists := files[expected]
		if !exists {
			t.Errorf("Expected file %s not found", expected)
			continue
		}

		// Verify file content is not empty
		if len(content) == 0 {
			t.Errorf("File %s has empty content", expected)
		}

		// Verify it's valid TOML content (basic check)
		contentStr := string(content)
		if !strings.Contains(contentStr, "[provider]") {
			t.Errorf("File %s doesn't appear to contain provider configuration", expected)
		}
	}
}

func TestGetEmbeddedProviderFiles_ContentValidation(t *testing.T) {
	files, err := GetEmbeddedProviderFiles()
	if err != nil {
		t.Fatalf("GetEmbeddedProviderFiles() failed: %v", err)
	}

	// Test that each file can be parsed as TOML
	for filename, content := range files {
		t.Run(filename, func(t *testing.T) {
			var config struct {
				Provider types.Provider `toml:"provider"`
			}

			// This should not fail for any embedded file
			_, err := parseTomlConfig(string(content), &config)
			if err != nil {
				t.Errorf("Failed to parse %s as TOML: %v", filename, err)
			}

			// Verify provider name is set
			if config.Provider.Name == "" {
				t.Errorf("Provider in %s has empty name", filename)
			}

			// Verify filename matches provider name
			expectedName := strings.TrimSuffix(filename, ".toml")
			if config.Provider.Name != expectedName {
				t.Errorf("Provider name %s doesn't match filename %s", config.Provider.Name, expectedName)
			}
		})
	}
}

func TestGetEmbeddedProviderFiles_ConsistencyWithLoadEmbeddedProviders(t *testing.T) {
	// Load providers using both methods
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	files, err := GetEmbeddedProviderFiles()
	if err != nil {
		t.Fatalf("GetEmbeddedProviderFiles() failed: %v", err)
	}

	// Verify that the number of providers matches the number of files
	if len(providers) != len(files) {
		t.Errorf("Number of providers (%d) doesn't match number of files (%d)", len(providers), len(files))
	}

	// Verify that each provider has a corresponding file
	for providerName := range providers {
		expectedFilename := providerName + ".toml"
		if _, exists := files[expectedFilename]; !exists {
			t.Errorf("Provider %s doesn't have corresponding file %s", providerName, expectedFilename)
		}
	}

	// Verify that each file has a corresponding provider
	for filename := range files {
		expectedProviderName := strings.TrimSuffix(filename, ".toml")
		if _, exists := providers[expectedProviderName]; !exists {
			t.Errorf("File %s doesn't have corresponding provider %s", filename, expectedProviderName)
		}
	}
}

func TestLoadEmbeddedProviders_SpecificProviderFeatures(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	// Test AUR helpers have proper characteristics
	aurHelpers := []string{"yay", "paru", "aura", "trizen"}
	for _, helper := range aurHelpers {
		if provider, exists := providers[helper]; exists {
			t.Run(helper+"_AUR_characteristics", func(t *testing.T) {
				// AUR helpers should detect Arch-based distributions
				hasArchDistro := false
				for _, distro := range provider.Detection.Distributions {
					if strings.Contains(distro, "arch") || strings.Contains(distro, "manjaro") {
						hasArchDistro = true
						break
					}
				}
				if !hasArchDistro {
					t.Errorf("AUR helper %s should support Arch-based distributions", helper)
				}

				// Should have install steps for building from AUR
				if len(provider.Install.Steps) == 0 {
					t.Errorf("AUR helper %s should have install steps", helper)
				}

				// Should use pacman-related detection files
				hasPackmanFiles := false
				for _, file := range provider.Detection.Files {
					if strings.Contains(file, "pacman") {
						hasPackmanFiles = true
						break
					}
				}
				if !hasPackmanFiles {
					t.Errorf("AUR helper %s should detect pacman-related files", helper)
				}
			})
		}
	}

	// Test paru is non-elevated while others are elevated
	if paru, exists := providers["paru"]; exists {
		if paru.Elevated {
			t.Error("paru should be non-elevated according to its design")
		}
	}

	// Test repository management for pacman-based providers
	pacmanProviders := []string{"pacman", "yay", "paru", "aura", "trizen", "pamac"}
	for _, providerName := range pacmanProviders {
		if provider, exists := providers[providerName]; exists {
			t.Run(providerName+"_repository", func(t *testing.T) {
				// Should have repository paths configured
				if provider.Repository.Paths.Sources == "" {
					t.Error("Provider should have repository sources path")
				}
				if provider.Repository.Paths.Keys == "" {
					t.Error("Provider should have repository keys path")
				}

				// Should have repository add/remove steps
				if len(provider.Repository.Add.Steps) == 0 {
					t.Error("Provider should have repository add steps")
				}
				if len(provider.Repository.Remove.Steps) == 0 {
					t.Error("Provider should have repository remove steps")
				}
			})
		}
	}
}

func TestLoadEmbeddedProviders_TemplateVariableUsage(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatalf("LoadEmbeddedProviders() failed: %v", err)
	}

	// Check for proper template variable usage in repository steps
	templateVars := []string{"{{ .Name }}", "{{ .URL }}", "{{ .KeyID }}", "{{ .KeyURL }}", "{{ .KeyPath }}"}

	for name, provider := range providers {
		t.Run(name+"_templates", func(t *testing.T) {
			// Check repository add steps for template variables
			for _, step := range provider.Repository.Add.Steps {
				if step.Content != "" {
					hasTemplateVar := false
					for _, tmplVar := range templateVars {
						if strings.Contains(step.Content, tmplVar) {
							hasTemplateVar = true
							break
						}
					}
					if !hasTemplateVar && strings.Contains(step.Content, "{{") {
						t.Logf("Repository add step in %s uses templates but not recognized ones: %s", name, step.Content)
					}
				}

				// Check args for template usage
				for _, arg := range step.Args {
					if strings.Contains(arg, "{{") && !containsValidTemplate(arg, templateVars) {
						t.Logf("Repository add step arg in %s uses unrecognized template: %s", name, arg)
					}
				}
			}

			// Check repository remove steps
			for _, step := range provider.Repository.Remove.Steps {
				if step.Content != "" && strings.Contains(step.Content, "{{") {
					hasValidTemplate := containsValidTemplate(step.Content, templateVars)
					if !hasValidTemplate {
						t.Logf("Repository remove step in %s uses unrecognized template: %s", name, step.Content)
					}
				}
			}
		})
	}
}

// Helper function to check if content contains valid template variables
func containsValidTemplate(content string, validTemplates []string) bool {
	for _, tmpl := range validTemplates {
		if strings.Contains(content, tmpl) {
			return true
		}
	}
	return false
}

// Helper function to parse TOML configuration
func parseTomlConfig(content string, config interface{}) (interface{}, error) {
	_, err := toml.Decode(content, config)
	return config, err
}
