package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestPackageImports(t *testing.T) {
	tempDir := t.TempDir()

	// Create test import file
	importFile := filepath.Join(tempDir, "base-packages.yaml")
	importContent := `packages:
  - names:
      - git
      - curl
      - vim
    action: install
    package_manager: apt
`
	if err := os.WriteFile(importFile, []byte(importContent), 0644); err != nil {
		t.Fatalf("Failed to create import file: %v", err)
	}

	// Create main package file with import
	mainFile := filepath.Join(tempDir, "packages.yaml")
	mainContent := `packages:
  - import: base-packages.yaml
  - names:
      - docker
      - kubernetes
    action: install
    package_manager: apt
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Read and process
	data, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    false,
				Profiles: []string{},
			},
		},
	}

	// Process packages (this will trigger import processing)
	err = ProcessPackages(data, nil, "yaml", osInfo, initConfig)

	// We expect this to fail because package managers aren't available
	// but we can verify the import was processed by checking the error doesn't mention import issues
	if err != nil {
		if containsString(err.Error(), "error reading import file") {
			t.Errorf("Import file reading failed: %v", err)
		}
		if containsString(err.Error(), "error unmarshaling import file") {
			t.Errorf("Import file unmarshaling failed: %v", err)
		}
	}
}

func TestCircularImportDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create file A that imports B
	fileA := filepath.Join(tempDir, "packages-a.yaml")
	contentA := `packages:
  - import: packages-b.yaml
  - names:
      - package-a
    action: install
`
	if err := os.WriteFile(fileA, []byte(contentA), 0644); err != nil {
		t.Fatalf("Failed to create file A: %v", err)
	}

	// Create file B that imports A (circular)
	fileB := filepath.Join(tempDir, "packages-b.yaml")
	contentB := `packages:
  - import: packages-a.yaml
  - names:
      - package-b
    action: install
`
	if err := os.WriteFile(fileB, []byte(contentB), 0644); err != nil {
		t.Fatalf("Failed to create file B: %v", err)
	}

	data, err := os.ReadFile(fileA)
	if err != nil {
		t.Fatalf("Failed to read file A: %v", err)
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    false,
				Profiles: []string{},
			},
		},
	}

	// Process should complete without infinite loop
	// Circular import should be detected and skipped
	err = ProcessPackages(data, nil, "yaml", osInfo, initConfig)

	// Should complete without hanging (circular detection works)
	if err != nil && containsString(err.Error(), "circular") {
		// This is actually OK - we detected it
		t.Logf("Circular import properly detected: %v", err)
	}
}

func TestMultipleImports(t *testing.T) {
	tempDir := t.TempDir()

	// Create import file 1
	import1 := filepath.Join(tempDir, "base.yaml")
	content1 := `packages:
  - names:
      - git
      - vim
    action: install
`
	if err := os.WriteFile(import1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create import1: %v", err)
	}

	// Create import file 2
	import2 := filepath.Join(tempDir, "dev.yaml")
	content2 := `packages:
  - names:
      - docker
      - kubernetes
    action: install
`
	if err := os.WriteFile(import2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create import2: %v", err)
	}

	// Create main file with multiple imports
	mainFile := filepath.Join(tempDir, "packages.yaml")
	mainContent := `packages:
  - import: base.yaml
  - import: dev.yaml
  - names:
      - custom-tool
    action: install
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	data, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    false,
				Profiles: []string{},
			},
		},
	}

	err = ProcessPackages(data, nil, "yaml", osInfo, initConfig)

	// Verify imports were processed (no import-related errors)
	if err != nil {
		if containsString(err.Error(), "error reading import file") {
			t.Errorf("Import file reading failed: %v", err)
		}
		if containsString(err.Error(), "error unmarshaling import file") {
			t.Errorf("Import file unmarshaling failed: %v", err)
		}
	}
}

func TestMissingImportFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create main file with import to non-existent file
	mainFile := filepath.Join(tempDir, "packages.yaml")
	mainContent := `packages:
  - import: non-existent.yaml
  - names:
      - package-a
    action: install
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	data, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    false,
				Profiles: []string{},
			},
		},
	}

	err = ProcessPackages(data, nil, "yaml", osInfo, initConfig)

	// Should fail with import error
	if err == nil {
		t.Error("Expected error for missing import file, got nil")
	} else if !containsString(err.Error(), "error reading import file") {
		t.Errorf("Expected 'error reading import file', got: %v", err)
	}
}

func TestRelativeImportPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create subdirectory structure
	subdir := filepath.Join(tempDir, "common")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create import file in subdirectory
	importFile := filepath.Join(subdir, "base.yaml")
	importContent := `packages:
  - names:
      - shared-package
    action: install
`
	if err := os.WriteFile(importFile, []byte(importContent), 0644); err != nil {
		t.Fatalf("Failed to create import file: %v", err)
	}

	// Create main file with relative import
	mainFile := filepath.Join(tempDir, "packages.yaml")
	mainContent := `packages:
  - import: common/base.yaml
  - names:
      - local-package
    action: install
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	data, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    false,
				Profiles: []string{},
			},
		},
	}

	err = ProcessPackages(data, nil, "yaml", osInfo, initConfig)

	// Verify relative import worked (no import-related errors)
	if err != nil {
		if containsString(err.Error(), "error reading import file") {
			t.Errorf("Relative import failed: %v", err)
		}
	}
}
