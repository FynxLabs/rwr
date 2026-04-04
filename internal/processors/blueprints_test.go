package processors

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestGetBlueprintRunOrder_DefaultOrder(t *testing.T) {
	// Test with nil order - should return default order
	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: nil,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{
		"packageManagers", "repositories", "packages", "ssh_keys",
		"files", "fonts", "services", "git", "scripts", "configuration",
	}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Errorf("Expected default order %v, got %v", expectedOrder, result)
	}
}

func TestGetBlueprintRunOrder_CustomStringOrder(t *testing.T) {
	// Test with custom string order
	customOrder := []interface{}{
		"packages",
		"services",
		"files",
		"scripts",
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: customOrder,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{"packages", "services", "files", "scripts"}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Errorf("Expected custom order %v, got %v", expectedOrder, result)
	}
}

func TestGetBlueprintRunOrder_CustomMapOrder(t *testing.T) {
	// Test with map order (processor with sub-configuration)
	customOrder := []interface{}{
		map[string]interface{}{
			"packages": map[string]interface{}{
				"source": "packages/",
			},
		},
		map[string]interface{}{
			"services": map[string]interface{}{
				"source": "services/",
			},
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: customOrder,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{"packages", "services"}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Errorf("Expected map order %v, got %v", expectedOrder, result)
	}
}

func TestGetBlueprintRunOrder_MixedOrder(t *testing.T) {
	// Test with mixed string and map order
	mixedOrder := []interface{}{
		"packages",
		map[string]interface{}{
			"services": map[string]interface{}{
				"source": "services/",
			},
		},
		"files",
		map[string]interface{}{
			"scripts": map[string]interface{}{
				"source": "scripts/",
			},
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: mixedOrder,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{"packages", "services", "files", "scripts"}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Errorf("Expected mixed order %v, got %v", expectedOrder, result)
	}
}

func TestGetBlueprintRunOrder_EmptyOrder(t *testing.T) {
	// Test with empty order slice - should return empty order (no processors to run)
	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: []interface{}{},
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the result is empty (length 0), handling nil vs empty slice difference
	if len(result) != 0 {
		t.Errorf("Expected empty order for empty slice, got %v with length %d", result, len(result))
	}
}

func TestGetBlueprintRunOrder_SingleItem(t *testing.T) {
	// Test with single item order
	singleOrder := []interface{}{"packages"}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: singleOrder,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedOrder := []string{"packages"}

	if !reflect.DeepEqual(result, expectedOrder) {
		t.Errorf("Expected single item order %v, got %v", expectedOrder, result)
	}
}

func TestGetBlueprintRunOrder_MultipleMapItems(t *testing.T) {
	// Test with multiple map items containing multiple processors each
	multiMapOrder := []interface{}{
		map[string]interface{}{
			"packages": map[string]interface{}{
				"source": "packages/",
			},
			"repositories": map[string]interface{}{
				"source": "repositories/",
			},
		},
		map[string]interface{}{
			"services": map[string]interface{}{
				"source": "services/",
			},
			"files": map[string]interface{}{
				"source": "files/",
			},
		},
	}

	initConfig := &types.InitConfig{
		Init: types.Init{
			Order: multiMapOrder,
		},
	}

	result, err := GetBlueprintRunOrder(initConfig)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should extract all processor keys from maps
	// Note: map iteration order is not guaranteed, so we check length and contents
	if len(result) != 4 {
		t.Errorf("Expected 4 processors, got %d: %v", len(result), result)
	}

	expectedProcessors := map[string]bool{
		"packages": true, "repositories": true, "services": true, "files": true,
	}

	for _, processor := range result {
		if !expectedProcessors[processor] {
			t.Errorf("Unexpected processor '%s' in result", processor)
		}
	}
}

// GetBlueprintFileOrder tests

func TestGetBlueprintFileOrder_DirectoryScan(t *testing.T) {
	// Create temp blueprint directory structure
	tempDir := t.TempDir()

	// Create processor directories with blueprint files
	dirs := []string{"packages", "services", "files"}
	for _, d := range dirs {
		dir := filepath.Join(tempDir, d)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		file := filepath.Join(dir, d+".yaml")
		if err := os.WriteFile(file, []byte("test: data"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	result, err := GetBlueprintFileOrder(tempDir, nil, false, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	// Should find files for packages, services, files processors
	for _, processor := range dirs {
		if files, ok := result[processor]; !ok {
			t.Errorf("Expected processor '%s' in file order", processor)
		} else if len(files) == 0 {
			t.Errorf("Expected files for processor '%s'", processor)
		}
	}
}

func TestGetBlueprintFileOrder_OrderedItems(t *testing.T) {
	tempDir := t.TempDir()

	// Create a packages directory with a file
	pkgDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "packages.yaml"), []byte("test"), 0644)

	// Create a services directory with a file
	svcDir := filepath.Join(tempDir, "services")
	os.MkdirAll(svcDir, 0755)
	os.WriteFile(filepath.Join(svcDir, "services.yaml"), []byte("test"), 0644)

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	order := []interface{}{"packages", "services"}
	result, err := GetBlueprintFileOrder(tempDir, order, true, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	if _, ok := result["packages"]; !ok {
		t.Error("Expected packages in file order")
	}
	if _, ok := result["services"]; !ok {
		t.Error("Expected services in file order")
	}
}

func TestGetBlueprintFileOrder_RunOnlyListed(t *testing.T) {
	tempDir := t.TempDir()

	// Create packages and services dirs
	for _, d := range []string{"packages", "services", "scripts"} {
		dir := filepath.Join(tempDir, d)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, d+".yaml"), []byte("test"), 0644)
	}

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	// Only list packages in order, with runOnlyListed=true
	order := []interface{}{"packages"}
	result, err := GetBlueprintFileOrder(tempDir, order, true, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	// Should only have packages, not services or scripts
	if _, ok := result["packages"]; !ok {
		t.Error("Expected packages in file order")
	}
	if _, ok := result["services"]; ok {
		t.Error("services should NOT be in file order when runOnlyListed=true")
	}
	if _, ok := result["scripts"]; ok {
		t.Error("scripts should NOT be in file order when runOnlyListed=true")
	}
}

func TestGetBlueprintFileOrder_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	result, err := GetBlueprintFileOrder(tempDir, nil, false, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty file order for empty directory, got %d processors", len(result))
	}
}

func TestGetBlueprintFileOrder_SingleFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a single file directly in packages/
	pkgDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "base.yaml"), []byte("test"), 0644)

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	// Reference the single file in order
	order := []interface{}{filepath.Join("packages", "base.yaml")}
	result, err := GetBlueprintFileOrder(tempDir, order, true, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	if _, ok := result["packages"]; !ok {
		t.Error("Expected packages processor for single file reference")
	}
}

func TestGetBlueprintFileOrder_MultipleFilesInProcessor(t *testing.T) {
	tempDir := t.TempDir()

	pkgDir := filepath.Join(tempDir, "packages")
	os.MkdirAll(pkgDir, 0755)
	os.WriteFile(filepath.Join(pkgDir, "base.yaml"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(pkgDir, "extra.yaml"), []byte("test"), 0644)

	config := &types.InitConfig{
		Init: types.Init{
			Format: "yaml",
		},
	}

	result, err := GetBlueprintFileOrder(tempDir, nil, false, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder failed: %v", err)
	}

	if files, ok := result["packages"]; !ok {
		t.Error("Expected packages in file order")
	} else if len(files) != 2 {
		t.Errorf("Expected 2 files for packages, got %d", len(files))
	}
}

// GetBlueprints tests

func TestGetBlueprints_NoGitOptions(t *testing.T) {
	// When no Git options are configured, should return the default location
	tempDir := t.TempDir()
	config := &types.InitConfig{
		Init: types.Init{
			Location: tempDir,
		},
	}

	result, err := GetBlueprints(config)
	if err != nil {
		t.Fatalf("GetBlueprints failed: %v", err)
	}

	if result != tempDir {
		t.Errorf("Expected location %s, got %s", tempDir, result)
	}
}

func TestGetBlueprints_NoGitOptions_EmptyLocation(t *testing.T) {
	config := &types.InitConfig{
		Init: types.Init{
			Location: "",
		},
	}

	result, err := GetBlueprints(config)
	if err != nil {
		t.Fatalf("GetBlueprints failed: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty location, got %s", result)
	}
}
