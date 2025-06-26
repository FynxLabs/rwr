package processors

import (
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
