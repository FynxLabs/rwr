package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadTestBlueprint loads a test blueprint file and replaces template variables
func loadTestBlueprint(t *testing.T, format, filename string, templateVars map[string]string) []byte {
	t.Helper()

	testdataPath := filepath.Join("testdata", "blueprints", format, filename+"."+format)
	content, err := os.ReadFile(testdataPath)
	if err != nil {
		t.Fatalf("Failed to read test blueprint %s: %v", testdataPath, err)
	}

	// Replace template variables
	contentStr := string(content)
	for key, value := range templateVars {
		contentStr = strings.ReplaceAll(contentStr, "{{."+key+"}}", value)
	}

	return []byte(contentStr)
}

// testFormats represents the file formats to test
var testFormats = []struct {
	name   string
	format string
}{
	{"YAML", "yaml"},
	{"JSON", "json"},
	{"TOML", "toml"},
}
