package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantIssues int
	}{
		{"empty value adds error", "", 1},
		{"non-empty value adds no error", "valid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			validateRequired(tt.value, "test.field", "/test.yaml", results, "fix it")
			if len(results.Issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d", len(results.Issues), tt.wantIssues)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	allowed := []string{"install", "remove", "update"}

	tests := []struct {
		name       string
		value      string
		wantIssues int
		wantMsg    string
	}{
		{"empty value adds error", "", 1, "Missing required field"},
		{"valid value adds no error", "install", 0, ""},
		{"invalid value adds error", "destroy", 1, "Invalid value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			validateEnum(tt.value, "test.action", allowed, "/test.yaml", results)
			if len(results.Issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d", len(results.Issues), tt.wantIssues)
			}
			if tt.wantMsg != "" && len(results.Issues) > 0 {
				if !contains(results.Issues[0].Message, tt.wantMsg) {
					t.Errorf("expected message containing %q, got %q", tt.wantMsg, results.Issues[0].Message)
				}
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantIssues int
	}{
		{"empty path adds no warning", "", 0},
		{"absolute path adds no warning", "/usr/bin/test", 0},
		{"tilde path adds no warning", "~/config", 0},
		{"relative path adds warning", "config/file", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			validatePath(tt.path, "test field", "/test.yaml", results)
			if len(results.Issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d", len(results.Issues), tt.wantIssues)
			}
		})
	}
}

func TestValidateImport(t *testing.T) {
	// Create temp dir with a valid import file
	tempDir, err := os.MkdirTemp("", "rwr_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a valid YAML import file
	validFile := filepath.Join(tempDir, "imported.yaml")
	os.WriteFile(validFile, []byte("packages:\n  - name: test\n    action: install\n"), 0644)

	tests := []struct {
		name         string
		importPath   string
		blueprintDir string
		isImport     bool
		wantIssues   int
	}{
		{"empty import returns false", "", tempDir, false, 0},
		{"valid import file", "imported.yaml", tempDir, true, 0},
		{"missing import file", "nonexistent.yaml", tempDir, true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &types.ValidationResults{}
			got := validateImport(tt.importPath, "test[0]", tt.blueprintDir, "/test.yaml", results, &types.PackagesData{})
			if got != tt.isImport {
				t.Errorf("got isImport=%v, want %v", got, tt.isImport)
			}
			if len(results.Issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d", len(results.Issues), tt.wantIssues)
			}
		})
	}
}

func TestValidateImport_CircularDetection(t *testing.T) {
	// Create temp dir with two files that import each other
	tempDir, err := os.MkdirTemp("", "rwr_circular_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// File A imports File B
	fileA := filepath.Join(tempDir, "a.yaml")
	os.WriteFile(fileA, []byte("packages:\n  - import: b.yaml\n"), 0644)

	// File B imports File A (circular)
	fileB := filepath.Join(tempDir, "b.yaml")
	os.WriteFile(fileB, []byte("packages:\n  - import: a.yaml\n"), 0644)

	results := &types.ValidationResults{}
	visited := make(map[string]bool)

	// Mark a.yaml as visited (simulating we started from a.yaml)
	absA, _ := filepath.Abs(fileA)
	visited[absA] = true

	// Now validate b.yaml's import of a.yaml - should detect circular
	got := validateImportWithVisited("a.yaml", "packages[0]", tempDir, fileB, results, &types.PackagesData{}, visited)
	if !got {
		t.Error("Expected validateImportWithVisited to return true for import")
	}

	// Should have a circular import error
	foundCircular := false
	for _, issue := range results.Issues {
		if contains(issue.Message, "Circular import") {
			foundCircular = true
			break
		}
	}
	if !foundCircular {
		t.Error("Expected circular import to be detected")
	}
}

func TestValidateImport_RecursiveValidation(t *testing.T) {
	// Create temp dir with a main file that imports a file with invalid content
	tempDir, err := os.MkdirTemp("", "rwr_recursive_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an imported file with a package missing required fields
	importFile := filepath.Join(tempDir, "imported.yaml")
	os.WriteFile(importFile, []byte("packages:\n  - name: \"\"\n    action: install\n"), 0644)

	results := &types.ValidationResults{}
	got := validateImport("imported.yaml", "packages[0]", tempDir, filepath.Join(tempDir, "main.yaml"), results, &types.PackagesData{})
	if !got {
		t.Error("Expected validateImport to return true for import")
	}

	// Should have validation errors from the imported content (empty name)
	foundNameError := false
	for _, issue := range results.Issues {
		if contains(issue.Message, "Missing required field") && contains(issue.Message, "name") {
			foundNameError = true
			break
		}
	}
	if !foundNameError {
		t.Error("Expected recursive validation to detect missing name in imported file")
	}
}

func TestValidateImport_NestedImports(t *testing.T) {
	// Create temp dir with chained imports: main -> a.yaml -> b.yaml
	tempDir, err := os.MkdirTemp("", "rwr_nested_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// b.yaml has actual packages
	fileB := filepath.Join(tempDir, "b.yaml")
	os.WriteFile(fileB, []byte("packages:\n  - name: vim\n    action: install\n"), 0644)

	// a.yaml imports b.yaml
	fileA := filepath.Join(tempDir, "a.yaml")
	os.WriteFile(fileA, []byte("packages:\n  - import: b.yaml\n"), 0644)

	results := &types.ValidationResults{}
	got := validateImport("a.yaml", "packages[0]", tempDir, filepath.Join(tempDir, "main.yaml"), results, &types.PackagesData{})
	if !got {
		t.Error("Expected validateImport to return true for import")
	}

	// Should have no errors (valid chain)
	errorCount := 0
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			errorCount++
		}
	}
	if errorCount > 0 {
		t.Errorf("Expected no errors for valid nested imports, got %d", errorCount)
		for _, issue := range results.Issues {
			t.Logf("  Issue: %s", issue.Message)
		}
	}
}

func TestValidateImport_SelfImportCircular(t *testing.T) {
	// Create a file that imports itself
	tempDir, err := os.MkdirTemp("", "rwr_self_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	selfFile := filepath.Join(tempDir, "self.yaml")
	os.WriteFile(selfFile, []byte("packages:\n  - import: self.yaml\n"), 0644)

	results := &types.ValidationResults{}
	// Simulate that we're validating self.yaml and it imports itself
	got := validateImport("self.yaml", "packages[0]", tempDir, selfFile, results, &types.PackagesData{})
	if !got {
		t.Error("Expected validateImport to return true for import")
	}

	// The recursive validation should detect the self-import as circular
	// since validateImportedContent will try to follow the import again
	// First call adds self.yaml to visited, recursive call should detect it
	foundCircular := false
	for _, issue := range results.Issues {
		if contains(issue.Message, "Circular import") {
			foundCircular = true
			break
		}
	}
	if !foundCircular {
		t.Error("Expected self-import circular to be detected")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
