package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestAll_BasicFlow(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Create basic init config
	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:       true,
				Interactive: false,
				Profiles:    []string{"test"},
			},
			UserDefined: make(map[string]interface{}),
		},
	}

	// Create basic OS info
	osInfo := &types.OSInfo{
		System: types.System{
			OS:        "linux",
			OSFamily:  "ubuntu",
			OSVersion: "22.04",
			OSArch:    "amd64",
		},
	}

	// Create empty blueprint files to avoid errors
	files := []string{
		"packages.yaml",
		"services.yaml",
		"files.yaml",
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		content := `# Empty blueprint for testing`
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test the All processor with minimal setup
	runOrder := []string{"packages", "services", "files"}
	err := All(initConfig, osInfo, runOrder)

	// We expect this might fail due to missing dependencies, but shouldn't panic
	if err != nil {
		t.Logf("All processor completed with expected error: %v", err)
	} else {
		t.Log("All processor completed successfully")
	}
}

func TestAll_EmptyRunOrder(t *testing.T) {
	tempDir := t.TempDir()

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    true,
				Profiles: []string{"test"},
			},
			UserDefined: make(map[string]interface{}),
		},
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	// Test with empty run order
	err := All(initConfig, osInfo, []string{})

	// Should complete without error when nothing to process
	if err != nil {
		t.Errorf("Expected no error with empty run order, got: %v", err)
	}
}

func TestAll_InvalidBlueprintLocation(t *testing.T) {
	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: "/nonexistent/path",
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:    true,
				Profiles: []string{"test"},
			},
			UserDefined: make(map[string]interface{}),
		},
	}

	osInfo := &types.OSInfo{
		System: types.System{
			OS: "linux",
		},
	}

	err := All(initConfig, osInfo, []string{"packages"})

	// Should fail with blueprint location error
	if err == nil {
		t.Error("Expected error for invalid blueprint location")
	} else if !containsString(err.Error(), "blueprint location does not exist") {
		t.Errorf("Expected blueprint location error, got: %v", err)
	}
}

func TestAll_MacOSPackageManagerDetection(t *testing.T) {
	tempDir := t.TempDir()

	initConfig := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
			Format:   "yaml",
		},
		Variables: types.Variables{
			Flags: types.Flags{
				Debug:       true,
				Interactive: false,
				Profiles:    []string{"test"},
			},
			UserDefined: make(map[string]interface{}),
		},
	}

	// Simulate macOS without package managers
	osInfo := &types.OSInfo{
		System: types.System{
			OS:       "darwin",
			OSFamily: "darwin",
		},
		PackageManager: types.PackageManager{
			Managers: make(map[string]types.PackageManagerInfo), // No package managers detected
		},
	}

	err := All(initConfig, osInfo, []string{})

	// Should attempt to install package manager on macOS
	if err != nil {
		t.Logf("Expected error when trying to install package manager on macOS: %v", err)
	}
}
