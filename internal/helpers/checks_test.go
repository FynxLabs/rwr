package helpers

import (
	"fmt"
	"testing"
)

func TestContains_ItemExists(t *testing.T) {
	slice := []string{"apple", "banana", "cherry", "date"}
	item := "banana"

	result := Contains(slice, item)

	if !result {
		t.Errorf("Expected Contains(%v, %s) to be true, got false", slice, item)
	}
}

func TestContains_ItemNotExists(t *testing.T) {
	slice := []string{"apple", "banana", "cherry", "date"}
	item := "grape"

	result := Contains(slice, item)

	if result {
		t.Errorf("Expected Contains(%v, %s) to be false, got true", slice, item)
	}
}

func TestContains_EmptySlice(t *testing.T) {
	slice := []string{}
	item := "anything"

	result := Contains(slice, item)

	if result {
		t.Errorf("Expected Contains(%v, %s) to be false, got true", slice, item)
	}
}

func TestContains_EmptyItem(t *testing.T) {
	slice := []string{"apple", "", "cherry"}
	item := ""

	result := Contains(slice, item)

	if !result {
		t.Errorf("Expected Contains(%v, %s) to be true, got false", slice, item)
	}
}

func TestContains_NilSlice(t *testing.T) {
	var slice []string
	item := "anything"

	result := Contains(slice, item)

	if result {
		t.Errorf("Expected Contains(nil, %s) to be false, got true", item)
	}
}

func TestContains_CaseSensitive(t *testing.T) {
	slice := []string{"Apple", "Banana", "Cherry"}

	// Should not match different case
	result1 := Contains(slice, "apple")
	if result1 {
		t.Errorf("Expected Contains(%v, %s) to be false (case sensitive), got true", slice, "apple")
	}

	// Should match exact case
	result2 := Contains(slice, "Apple")
	if !result2 {
		t.Errorf("Expected Contains(%v, %s) to be true (exact case), got false", slice, "Apple")
	}
}

func TestContains_DuplicateItems(t *testing.T) {
	slice := []string{"apple", "banana", "apple", "cherry"}
	item := "apple"

	result := Contains(slice, item)

	if !result {
		t.Errorf("Expected Contains(%v, %s) to be true, got false", slice, item)
	}
}

func TestContains_SingleItem(t *testing.T) {
	slice := []string{"onlyitem"}

	// Should find the single item
	result1 := Contains(slice, "onlyitem")
	if !result1 {
		t.Errorf("Expected Contains(%v, %s) to be true, got false", slice, "onlyitem")
	}

	// Should not find different item
	result2 := Contains(slice, "otheritem")
	if result2 {
		t.Errorf("Expected Contains(%v, %s) to be false, got true", slice, "otheritem")
	}
}

func TestContains_SpecialCharacters(t *testing.T) {
	slice := []string{"item-with-dashes", "item_with_underscores", "item.with.dots", "item/with/slashes"}

	testCases := []struct {
		item     string
		expected bool
	}{
		{"item-with-dashes", true},
		{"item_with_underscores", true},
		{"item.with.dots", true},
		{"item/with/slashes", true},
		{"item-with-underscores", false}, // Different from actual item
		{"nonexistent", false},
	}

	for _, tc := range testCases {
		result := Contains(slice, tc.item)
		if result != tc.expected {
			t.Errorf("Expected Contains(%v, %s) to be %v, got %v", slice, tc.item, tc.expected, result)
		}
	}
}

// Benchmark test to ensure good performance
func BenchmarkContains_SmallSlice(b *testing.B) {
	slice := []string{"item1", "item2", "item3", "item4", "item5"}
	item := "item3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains(slice, item)
	}
}

func BenchmarkContains_LargeSlice(b *testing.B) {
	slice := make([]string, 1000)
	for i := range slice {
		slice[i] = fmt.Sprintf("item%d", i)
	}
	item := "item500"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Contains(slice, item)
	}
}
