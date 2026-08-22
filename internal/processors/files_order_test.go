package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestProcessFiles_DirectoryBaselinePrecedesFileMetadata(t *testing.T) {
	blueprintDir := t.TempDir()
	sourceDir := filepath.Join(blueprintDir, "config")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "tool.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetRoot := t.TempDir()
	targetFile := filepath.Join(targetRoot, "config", "tool.sh")
	blueprint := []byte(`
directories:
  - action: copy
    name: config
    source: .
    target: ` + targetRoot + `
files:
  - action: chmod
    name: tool.sh
    target: ` + targetFile + `
    mode: "0755"
`)
	config := &types.InitConfig{Variables: types.Variables{UserDefined: map[string]interface{}{}}}

	if err := ProcessFiles(blueprint, blueprintDir, types.FormatYAML, &types.OSInfo{}, config); err != nil {
		t.Fatalf("ProcessFiles: %v", err)
	}
	info, err := os.Stat(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("tool mode = %04o, want 0755 after directory copy then chmod", got)
	}
}
