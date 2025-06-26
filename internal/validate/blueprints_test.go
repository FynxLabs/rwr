package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestFindInitFile_ValidInitFiles(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "rwr_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases for different init file extensions
	testCases := []struct {
		filename string
		expected bool
	}{
		{"init.yaml", true},
		{"init.yml", true},
		{"init.json", true},
		{"init.toml", true},
		{"other.yaml", false}, // Should not be found
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			// Create the test file
			testFile := filepath.Join(tempDir, tc.filename)
			err := os.WriteFile(testFile, []byte("test: content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := findInitFile(tempDir)

			if tc.expected {
				if result == "" {
					t.Errorf("Expected to find init file %s, got empty string", tc.filename)
				}
				if result != testFile {
					t.Errorf("Expected to find %s, got %s", testFile, result)
				}
			}

			// Clean up for next test
			os.Remove(testFile)
		})
	}
}

func TestFindInitFile_NoInitFile(t *testing.T) {
	// Create a temporary directory with no init files
	tempDir, err := os.MkdirTemp("", "rwr_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some non-init files
	os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "other.json"), []byte("test"), 0644)

	result := findInitFile(tempDir)

	if result != "" {
		t.Errorf("Expected empty string when no init file exists, got %s", result)
	}
}

func TestFindInitFile_MultipleInitFiles(t *testing.T) {
	// Create a temporary directory with multiple init files
	tempDir, err := os.MkdirTemp("", "rwr_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple init files
	initFiles := []string{"init.json", "init.yaml", "init.yml", "init.toml"}
	for _, filename := range initFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	result := findInitFile(tempDir)

	// Should find one of them (the function checks in a specific order)
	if result == "" {
		t.Error("Expected to find an init file when multiple exist")
	}

	// Verify the found file actually exists
	if _, err := os.Stat(result); os.IsNotExist(err) {
		t.Errorf("Found init file %s does not exist", result)
	}
}

func TestFindInitFile_EmptyDirectory(t *testing.T) {
	// Create empty temporary directory
	tempDir, err := os.MkdirTemp("", "rwr_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	result := findInitFile(tempDir)

	if result != "" {
		t.Errorf("Expected empty string for empty directory, got %s", result)
	}
}

func TestFindInitFile_NonExistentDirectory(t *testing.T) {
	// Test with a directory that doesn't exist
	nonExistentDir := "/path/that/definitely/does/not/exist"

	result := findInitFile(nonExistentDir)

	// Should return empty string, not crash
	if result != "" {
		t.Errorf("Expected empty string for non-existent directory, got %s", result)
	}
}

func TestValidateInitFile_NonExistentFile(t *testing.T) {
	nonExistentFile := "/path/that/does/not/exist/init.yaml"

	results := &types.ValidationResults{}
	_, _ = validateInitFile(nonExistentFile, results)

	// Should add validation error for file read error
	foundReadError := false
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			foundReadError = true
			break
		}
	}

	if !foundReadError {
		t.Error("Expected validation error for non-existent file")
	}
}

// Test helper functions
func TestAddIssue_AddsCorrectly(t *testing.T) {
	results := &types.ValidationResults{}

	AddIssue(results, types.ValidationError, "Test error message", "/test/file.yaml", 10, "Fix the issue")

	if len(results.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(results.Issues))
	}

	issue := results.Issues[0]
	if issue.Severity != types.ValidationError {
		t.Errorf("Expected ValidationError severity, got %v", issue.Severity)
	}

	if issue.Message != "Test error message" {
		t.Errorf("Expected 'Test error message', got '%s'", issue.Message)
	}

	if issue.File != "/test/file.yaml" {
		t.Errorf("Expected '/test/file.yaml', got '%s'", issue.File)
	}

	if issue.Line != 10 {
		t.Errorf("Expected line 10, got %d", issue.Line)
	}

	if issue.Suggestion != "Fix the issue" {
		t.Errorf("Expected 'Fix the issue', got '%s'", issue.Suggestion)
	}
}

func TestAddIssue_MultipleIssues(t *testing.T) {
	results := &types.ValidationResults{}

	AddIssue(results, types.ValidationError, "First error", "/test/file1.yaml", 1, "Fix first")
	AddIssue(results, types.ValidationWarning, "Second warning", "/test/file2.yaml", 2, "Fix second")
	AddIssue(results, types.ValidationInfo, "Third info", "/test/file3.yaml", 3, "Fix third")

	if len(results.Issues) != 3 {
		t.Errorf("Expected 3 issues, got %d", len(results.Issues))
	}

	// Verify each issue
	expectedSeverities := []types.ValidationSeverity{types.ValidationError, types.ValidationWarning, types.ValidationInfo}
	for i, expectedSeverity := range expectedSeverities {
		if results.Issues[i].Severity != expectedSeverity {
			t.Errorf("Expected issue %d to be %v, got %v", i, expectedSeverity, results.Issues[i].Severity)
		}
	}
}
