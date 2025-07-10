package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestInitialize_LocalYAMLFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test init file
	initContent := `
blueprints:
  location: "./blueprints"
  format: "yaml"
  order:
    - packages
    - services
    - files

variables:
  test_var: "test_value"
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	// Create blueprints directory
	blueprintsDir := filepath.Join(tempDir, "blueprints")
	if err := os.MkdirAll(blueprintsDir, 0755); err != nil {
		t.Fatalf("Failed to create blueprints directory: %v", err)
	}

	flags := types.Flags{
		Debug:    true,
		Profiles: []string{"test"},
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify config was loaded correctly
	if config.Init.Format != "yaml" {
		t.Errorf("Expected format 'yaml', got '%s'", config.Init.Format)
	}

	if len(config.Init.Order) != 3 {
		t.Errorf("Expected 3 order items, got %d", len(config.Init.Order))
	}

	if config.Variables.Flags.Debug != true {
		t.Error("Expected debug flag to be true")
	}

	if config.Variables.User.Username == "" {
		t.Error("Expected username to be populated")
	}
}

func TestInitialize_TOMLFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test TOML init file
	initContent := `
[blueprints]
location = "./blueprints"
format = "toml"
order = ["packages", "files"]

[variables]
env = "test"
`

	initFile := filepath.Join(tempDir, "init.toml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: false,
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// TOML should be converted to YAML internally
	if config.Init.Format != "toml" {
		t.Errorf("Expected format 'toml', got '%s'", config.Init.Format)
	}

	if len(config.Init.Order) != 2 {
		t.Errorf("Expected 2 order items, got %d", len(config.Init.Order))
	}
}

func TestInitialize_WithGitRepository(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test init file with git config
	initContent := `
blueprints:
  location: "./blueprints"
  format: "yaml"
  git:
    url: "https://github.com/test/repo.git"
    target: "` + filepath.Join(tempDir, "git-blueprints") + `"
    update: true
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: true,
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify git configuration
	if config.Init.Git == nil {
		t.Fatal("Expected git configuration to be set")
	}

	if config.Init.Git.URL != "https://github.com/test/repo.git" {
		t.Errorf("Expected git URL 'https://github.com/test/repo.git', got '%s'", config.Init.Git.URL)
	}

	if !config.Init.Git.Update {
		t.Error("Expected git update to be true")
	}
}

func TestInitialize_MissingFile(t *testing.T) {
	flags := types.Flags{
		Debug: true,
	}

	_, err := Initialize("/nonexistent/init.yaml", flags)

	if err == nil {
		t.Error("Expected error for missing init file")
	}

	if !containsString(err.Error(), "init file not found") {
		t.Errorf("Expected 'init file not found' error, got: %v", err)
	}
}

func TestInitialize_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid YAML
	initContent := `
blueprints:
  location: "./blueprints"
  format: "yaml
  # Missing closing quote - invalid YAML
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: true,
	}

	_, err := Initialize(initFile, flags)

	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestInitialize_TemplateVariables(t *testing.T) {
	tempDir := t.TempDir()

	// Create init file with template variables
	initContent := `
blueprints:
  location: "{{ .User.Home }}/blueprints"
  format: "yaml"

variables:
  user_home: "{{ .User.Home }}"
  username: "{{ .User.Username }}"
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: true,
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Variables should be populated with current user info
	if config.Variables.User.Username == "" {
		t.Error("Expected username to be populated")
	}

	if config.Variables.User.Home == "" {
		t.Error("Expected home directory to be populated")
	}
}

func TestInitialize_EnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()

	// Set test environment variable
	os.Setenv("RWR_TEST_VAR", "test_value")
	defer os.Unsetenv("RWR_TEST_VAR")

	initContent := `
blueprints:
  location: "./blueprints"
  format: "yaml"
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: true,
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Environment variable should be included in user-defined variables
	if config.Variables.UserDefined["TEST_VAR"] != "test_value" {
		t.Errorf("Expected RWR_TEST_VAR to be in user-defined variables, got: %v", config.Variables.UserDefined)
	}
}

func TestInitialize_RelativePaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create init file with relative paths
	initContent := `
blueprints:
  location: "./sub/blueprints"
  format: "yaml"
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		t.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: true,
	}

	config, err := Initialize(initFile, flags)

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Location should be resolved relative to init file
	expectedLocation := filepath.Join(tempDir, "sub", "blueprints")
	if config.Init.Location != expectedLocation {
		t.Errorf("Expected location '%s', got '%s'", expectedLocation, config.Init.Location)
	}
}

// BenchmarkInitialize tests the performance of initialization
func BenchmarkInitialize(b *testing.B) {
	tempDir := b.TempDir()

	initContent := `
blueprints:
  location: "./blueprints"
  format: "yaml"
  order:
    - packages
    - services
    - files
`

	initFile := filepath.Join(tempDir, "init.yaml")
	if err := os.WriteFile(initFile, []byte(initContent), 0644); err != nil {
		b.Fatalf("Failed to create test init file: %v", err)
	}

	flags := types.Flags{
		Debug: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Initialize(initFile, flags)
		if err != nil {
			b.Fatalf("Initialize failed: %v", err)
		}
	}
}
