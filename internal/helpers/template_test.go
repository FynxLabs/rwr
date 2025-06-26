package helpers

import (
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestResolveTemplate_BasicVariableSubstitution(t *testing.T) {
	templateData := []byte("Hello {{.User.username}}, your home is {{.User.home}}")

	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
			Home:     "/home/testuser",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expected := "Hello testuser, your home is /home/testuser"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_SystemVariables(t *testing.T) {
	templateData := []byte("OS: {{.System.os}}, Arch: {{.System.osArch}}")

	variables := types.Variables{
		System: types.System{
			OS:     "linux",
			OSArch: "amd64",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expected := "OS: linux, Arch: amd64"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_FlagsVariables(t *testing.T) {
	templateData := []byte("Debug: {{.Flags.debug}}, Interactive: {{.Flags.interactive}}")

	variables := types.Variables{
		Flags: types.Flags{
			Debug:       true,
			Interactive: false,
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expected := "Debug: true, Interactive: false"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_UserDefinedVariables(t *testing.T) {
	templateData := []byte("Custom: {{.UserDefined.CUSTOM_VAR}}, App: {{.UserDefined.APP_NAME}}")

	variables := types.Variables{
		UserDefined: map[string]interface{}{
			"CUSTOM_VAR": "custom_value",
			"APP_NAME":   "myapp",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expected := "Custom: custom_value, App: myapp"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_MixedVariables(t *testing.T) {
	templateData := []byte(`User: {{.User.username}}
OS: {{.System.os}}
Debug: {{.Flags.debug}}
Custom: {{.UserDefined.CUSTOM}}`)

	variables := types.Variables{
		User: types.UserInfo{
			Username: "admin",
		},
		System: types.System{
			OS: "windows",
		},
		Flags: types.Flags{
			Debug: true,
		},
		UserDefined: map[string]interface{}{
			"CUSTOM": "value123",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expectedLines := []string{
		"User: admin",
		"OS: windows",
		"Debug: true",
		"Custom: value123",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected result to contain '%s', got: %s", expected, resultStr)
		}
	}
}

func TestResolveTemplate_NoTemplate(t *testing.T) {
	// Test with data that has no template variables
	templateData := []byte("This is just plain text with no variables")

	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultStr := string(result)
	expected := "This is just plain text with no variables"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_EmptyTemplate(t *testing.T) {
	templateData := []byte("")

	variables := types.Variables{}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error for empty template, got: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty template, got: %s", string(result))
	}
}

func TestResolveTemplate_InvalidTemplate(t *testing.T) {
	// Test with malformed template syntax
	templateData := []byte("Hello {{.User.Username") // Missing closing braces

	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err == nil {
		t.Fatal("Expected error for malformed template, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result for invalid template, got: %s", string(result))
	}
}

func TestResolveTemplate_MissingVariable(t *testing.T) {
	// Test with template that references non-existent variable
	templateData := []byte("Hello {{.User.nonExistentField}}")

	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	// Should handle missing variables gracefully with "missingkey=invalid" option
	if err != nil {
		t.Fatalf("Expected no error for missing variable (should use invalid option), got: %v", err)
	}

	resultStr := string(result)
	// Should contain some indication of invalid/missing value
	if resultStr == "" {
		t.Error("Expected some result even with missing variable")
	}
}

func TestResolveTemplate_SpecialCharacters(t *testing.T) {
	templateData := []byte("Special chars: {{.UserDefined.SPECIAL}}")

	variables := types.Variables{
		UserDefined: map[string]interface{}{
			"SPECIAL": "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error with special characters, got: %v", err)
	}

	resultStr := string(result)
	expected := "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_UnicodeCharacters(t *testing.T) {
	templateData := []byte("Unicode: {{.UserDefined.UNICODE}}")

	variables := types.Variables{
		UserDefined: map[string]interface{}{
			"UNICODE": "ñáéíóú你好世界🌍",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error with unicode characters, got: %v", err)
	}

	resultStr := string(result)
	expected := "Unicode: ñáéíóú你好世界🌍"

	if resultStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, resultStr)
	}
}

func TestResolveTemplate_MultilineTemplate(t *testing.T) {
	templateData := []byte(`# Configuration for {{.User.username}}
home_directory: {{.User.home}}
shell: {{.User.shell}}
# System info
os: {{.System.os}}
arch: {{.System.osArch}}`)

	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
			Home:     "/home/testuser",
			Shell:    "/bin/bash",
		},
		System: types.System{
			OS:     "linux",
			OSArch: "amd64",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error for multiline template, got: %v", err)
	}

	resultStr := string(result)

	// Check that all substitutions were made
	expectedSubstitutions := []string{
		"testuser",
		"/home/testuser",
		"/bin/bash",
		"linux",
		"amd64",
	}

	for _, expected := range expectedSubstitutions {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected result to contain '%s', got: %s", expected, resultStr)
		}
	}

	// Check that template structure is preserved
	if !strings.Contains(resultStr, "# Configuration for") {
		t.Error("Expected multiline structure to be preserved")
	}
}

func TestResolveTemplate_NumericValues(t *testing.T) {
	templateData := []byte("Count: {{.UserDefined.COUNT}}, Size: {{.UserDefined.SIZE}}")

	variables := types.Variables{
		UserDefined: map[string]interface{}{
			"COUNT": 42,
			"SIZE":  1024.5,
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error with numeric values, got: %v", err)
	}

	resultStr := string(result)

	// Should convert numbers to strings
	if !strings.Contains(resultStr, "42") {
		t.Error("Expected integer value to be converted to string")
	}

	if !strings.Contains(resultStr, "1024.5") {
		t.Error("Expected float value to be converted to string")
	}
}

func TestResolveTemplate_AllUserInfoFields(t *testing.T) {
	templateData := []byte(`Username: {{.User.username}}
FirstName: {{.User.firstName}}
LastName: {{.User.lastName}}
FullName: {{.User.fullName}}
GroupName: {{.User.groupName}}
Home: {{.User.home}}
Shell: {{.User.shell}}`)

	variables := types.Variables{
		User: types.UserInfo{
			Username:  "jdoe",
			FirstName: "John",
			LastName:  "Doe",
			FullName:  "John Doe",
			GroupName: "users",
			Home:      "/home/jdoe",
			Shell:     "/bin/zsh",
		},
	}

	result, err := ResolveTemplate(templateData, variables)

	if err != nil {
		t.Fatalf("Expected no error for all user fields, got: %v", err)
	}

	resultStr := string(result)

	// Check all fields are substituted correctly
	expectedFields := map[string]string{
		"Username: jdoe":     "jdoe",
		"FirstName: John":    "John",
		"LastName: Doe":      "Doe",
		"FullName: John Doe": "John Doe",
		"GroupName: users":   "users",
		"Home: /home/jdoe":   "/home/jdoe",
		"Shell: /bin/zsh":    "/bin/zsh",
	}

	for expected, _ := range expectedFields {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected result to contain '%s', got: %s", expected, resultStr)
		}
	}
}

// Benchmark tests
func BenchmarkResolveTemplate_Simple(b *testing.B) {
	templateData := []byte("Hello {{.User.username}}")
	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveTemplate(templateData, variables)
	}
}

func BenchmarkResolveTemplate_Complex(b *testing.B) {
	templateData := []byte(`
User: {{.User.username}}
Home: {{.User.home}}
OS: {{.System.os}}
Arch: {{.System.osArch}}
Debug: {{.Flags.debug}}
Custom1: {{.UserDefined.VAR1}}
Custom2: {{.UserDefined.VAR2}}
Custom3: {{.UserDefined.VAR3}}
`)
	variables := types.Variables{
		User: types.UserInfo{
			Username: "testuser",
			Home:     "/home/testuser",
		},
		System: types.System{
			OS:     "linux",
			OSArch: "amd64",
		},
		Flags: types.Flags{
			Debug: true,
		},
		UserDefined: map[string]interface{}{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveTemplate(templateData, variables)
	}
}
