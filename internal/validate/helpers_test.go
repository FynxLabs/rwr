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
