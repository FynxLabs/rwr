package helpers

import (
	"testing"
)

// Test struct for unmarshaling tests
type TestBlueprint struct {
	Format   string   `yaml:"format" json:"format" toml:"format"`
	Location string   `yaml:"location" json:"location" toml:"location"`
	Order    []string `yaml:"order" json:"order" toml:"order"`
}

func TestUnmarshalBlueprint_YAML_Valid(t *testing.T) {
	yamlData := []byte(`
format: "yaml"
location: "/test/path"
order:
  - "packages"
  - "repositories"
  - "services"
`)

	var result TestBlueprint
	err := UnmarshalBlueprint(yamlData, "yaml", &result)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Format != "yaml" {
		t.Errorf("Expected format 'yaml', got '%s'", result.Format)
	}

	if result.Location != "/test/path" {
		t.Errorf("Expected location '/test/path', got '%s'", result.Location)
	}

	expectedOrder := []string{"packages", "repositories", "services"}
	if len(result.Order) != len(expectedOrder) {
		t.Errorf("Expected order length %d, got %d", len(expectedOrder), len(result.Order))
	}

	for i, expected := range expectedOrder {
		if i >= len(result.Order) || result.Order[i] != expected {
			t.Errorf("Expected order[%d] = '%s', got '%s'", i, expected, result.Order[i])
		}
	}
}

func TestUnmarshalBlueprint_JSON_Valid(t *testing.T) {
	jsonData := []byte(`{
	"format": "json",
	"location": "/test/json/path",
	"order": ["packages", "files", "scripts"]
}`)

	var result TestBlueprint
	err := UnmarshalBlueprint(jsonData, "json", &result)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", result.Format)
	}

	if result.Location != "/test/json/path" {
		t.Errorf("Expected location '/test/json/path', got '%s'", result.Location)
	}

	expectedOrder := []string{"packages", "files", "scripts"}
	if len(result.Order) != len(expectedOrder) {
		t.Errorf("Expected order length %d, got %d", len(expectedOrder), len(result.Order))
	}
}

func TestUnmarshalBlueprint_TOML_Valid(t *testing.T) {
	tomlData := []byte(`
format = "toml"
location = "/test/toml/path"
order = ["services", "git", "users"]
`)

	var result TestBlueprint
	err := UnmarshalBlueprint(tomlData, "toml", &result)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Format != "toml" {
		t.Errorf("Expected format 'toml', got '%s'", result.Format)
	}

	if result.Location != "/test/toml/path" {
		t.Errorf("Expected location '/test/toml/path', got '%s'", result.Location)
	}

	expectedOrder := []string{"services", "git", "users"}
	if len(result.Order) != len(expectedOrder) {
		t.Errorf("Expected order length %d, got %d", len(expectedOrder), len(result.Order))
	}
}

func TestUnmarshalBlueprint_InvalidFormat(t *testing.T) {
	testData := []byte(`some data`)
	var result TestBlueprint

	err := UnmarshalBlueprint(testData, "xml", &result)

	if err == nil {
		t.Fatal("Expected error for unsupported format, got nil")
	}

	expectedErrorMsg := "unsupported blueprint format: xml"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestUnmarshalBlueprint_MalformedYAML(t *testing.T) {
	malformedYAML := []byte(`
format: "yaml"
location: "/test/path"
order:
  - "packages"
  - invalid: yaml: structure
`)

	var result TestBlueprint
	err := UnmarshalBlueprint(malformedYAML, "yaml", &result)

	if err == nil {
		t.Fatal("Expected error for malformed YAML, got nil")
	}

	// Check that it's a YAML unmarshaling error
	if err.Error() == "" {
		t.Error("Expected non-empty error message for malformed YAML")
	}
}

func TestUnmarshalBlueprint_MalformedJSON(t *testing.T) {
	malformedJSON := []byte(`{
	"format": "json",
	"location": "/test/path",
	"order": ["packages" missing_comma "files"]
}`)

	var result TestBlueprint
	err := UnmarshalBlueprint(malformedJSON, "json", &result)

	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}

	// Check that it's a JSON unmarshaling error
	if err.Error() == "" {
		t.Error("Expected non-empty error message for malformed JSON")
	}
}

func TestUnmarshalBlueprint_MalformedTOML(t *testing.T) {
	malformedTOML := []byte(`
format = "toml"
location = "/test/path"
order = ["packages" "missing_comma" "files"]
`)

	var result TestBlueprint
	err := UnmarshalBlueprint(malformedTOML, "toml", &result)

	if err == nil {
		t.Fatal("Expected error for malformed TOML, got nil")
	}

	// Check that it's a TOML unmarshaling error
	if err.Error() == "" {
		t.Error("Expected non-empty error message for malformed TOML")
	}
}

func TestUnmarshalBlueprint_EmptyData(t *testing.T) {
	emptyData := []byte(``)
	var result TestBlueprint

	// Test with YAML format (should handle empty gracefully)
	err := UnmarshalBlueprint(emptyData, "yaml", &result)
	if err != nil {
		t.Errorf("Expected no error for empty YAML data, got: %v", err)
	}

	// Test with JSON format (should error on empty)
	err = UnmarshalBlueprint(emptyData, "json", &result)
	if err == nil {
		t.Error("Expected error for empty JSON data, got nil")
	}
}

func TestUnmarshalBlueprint_AlternativeFormats(t *testing.T) {
	yamlData := []byte(`format: "test"`)
	var result TestBlueprint

	// Test various format string variations
	testCases := []string{".yaml", ".yml", "yml", ".json", ".toml"}

	for _, format := range testCases {
		err := UnmarshalBlueprint(yamlData, format, &result)
		if format == ".json" || format == ".toml" {
			// These should error since yamlData is YAML format
			if err == nil {
				t.Errorf("Expected error for format '%s' with YAML data, got nil", format)
			}
		} else {
			// These should work since yamlData is YAML format
			if err != nil {
				t.Errorf("Expected no error for format '%s' with YAML data, got: %v", format, err)
			}
		}
	}
}
