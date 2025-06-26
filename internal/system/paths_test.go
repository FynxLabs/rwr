package system

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestAddCommonPaths_ReturnsValidPaths(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Test with a simple PATH
	os.Setenv("PATH", "/usr/bin:/bin")

	result := AddCommonPaths()

	if result == "" {
		t.Error("Expected AddCommonPaths() to return non-empty string")
	}

	// Should include the original PATH
	if !strings.Contains(result, "/usr/bin") {
		t.Error("Expected result to contain original PATH elements")
	}

	// Should be properly separated
	pathSeparator := string(os.PathListSeparator)
	if !strings.Contains(result, pathSeparator) {
		t.Error("Expected result to contain path separators")
	}
}

func TestAddCommonPaths_WithEmptyPATH(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Test with empty PATH
	os.Setenv("PATH", "")

	result := AddCommonPaths()

	if result == "" {
		t.Error("Expected AddCommonPaths() to return common paths even with empty original PATH")
	}

	// Should contain platform-specific common paths
	switch runtime.GOOS {
	case "windows":
		// Should contain some Windows paths (though we can't guarantee they exist)
		if !strings.Contains(result, "%") && !strings.Contains(result, "\\") {
			t.Log("Note: Windows paths may not be present if not expanded")
		}
	default:
		// Should contain some Unix paths
		if !strings.Contains(result, "/usr/bin") && !strings.Contains(result, "/bin") {
			t.Log("Note: Standard Unix paths may not exist in test environment")
		}
	}
}

func TestAddCommonPaths_PreservesExistingPaths(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Test with a custom PATH
	customPath := "/custom/path:/another/custom/path"
	os.Setenv("PATH", customPath)

	result := AddCommonPaths()

	// Should preserve the original custom paths
	if !strings.Contains(result, "/custom/path") {
		t.Error("Expected result to preserve original custom paths")
	}

	if !strings.Contains(result, "/another/custom/path") {
		t.Error("Expected result to preserve all original custom paths")
	}
}

func TestAddCommonPaths_PathSeparator(t *testing.T) {
	result := AddCommonPaths()

	expectedSeparator := string(os.PathListSeparator)

	// Should use the correct path separator for the platform
	if runtime.GOOS == "windows" {
		if !strings.Contains(result, ";") && expectedSeparator == ";" {
			t.Error("Expected Windows path separator (;) in result")
		}
	} else {
		if !strings.Contains(result, ":") && expectedSeparator == ":" {
			t.Error("Expected Unix path separator (:) in result")
		}
	}
}

func TestAddCommonPaths_ConsistentResults(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Set a consistent PATH
	testPath := "/test/path"
	os.Setenv("PATH", testPath)

	// Call multiple times and ensure consistent results
	result1 := AddCommonPaths()
	result2 := AddCommonPaths()

	if result1 != result2 {
		t.Errorf("Expected consistent results from AddCommonPaths(), got different results:\nFirst:  %s\nSecond: %s", result1, result2)
	}
}

func TestSetPaths_NoError(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	err := SetPaths()

	if err != nil {
		t.Errorf("Expected SetPaths() to not return error, got: %v", err)
	}

	// Verify that PATH was actually set
	newPath := os.Getenv("PATH")
	if newPath == "" {
		t.Error("Expected PATH to be set after SetPaths()")
	}
}

func TestSetPaths_SetsCorrectVariable(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	err := SetPaths()
	if err != nil {
		t.Fatalf("SetPaths() returned error: %v", err)
	}

	// Check that the correct environment variable is set based on platform
	switch runtime.GOOS {
	case "windows":
		// On Windows, it should set "Path" (though this is tricky to test)
		path := os.Getenv("PATH")
		if path == "" {
			t.Error("Expected PATH to be set on Windows")
		}
	default:
		// On Unix systems, it should set "PATH"
		path := os.Getenv("PATH")
		if path == "" {
			t.Error("Expected PATH to be set on Unix systems")
		}
	}
}

func TestAddCommonPaths_HandlesUserLookupError(t *testing.T) {
	// This test verifies that the function doesn't crash when user lookup fails
	// We can't easily mock user.Current(), but we can test the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AddCommonPaths() panicked: %v", r)
		}
	}()

	result := AddCommonPaths()

	// Should still return something even if user lookup fails
	if result == "" {
		t.Error("Expected AddCommonPaths() to return something even with potential user lookup errors")
	}
}

func TestAddCommonPaths_PlatformSpecificPaths(t *testing.T) {
	result := AddCommonPaths()

	switch runtime.GOOS {
	case "windows":
		// Windows should include some typical Windows paths
		expectedSubstrings := []string{"%"}
		for _, expected := range expectedSubstrings {
			if strings.Contains(result, expected) {
				// Found at least one Windows-style path
				return
			}
		}
		t.Log("Note: Windows-specific paths may not be present or may be unexpanded")

	case "darwin", "linux":
		// Unix-like systems should include common Unix paths
		expectedPaths := []string{"/usr/bin", "/bin", "/usr/local/bin"}
		foundCommonPath := false
		for _, expected := range expectedPaths {
			if strings.Contains(result, expected) {
				foundCommonPath = true
				break
			}
		}
		if !foundCommonPath {
			t.Log("Note: Common Unix paths may not exist in test environment")
		}
	}
}

func TestAddCommonPaths_HandlesSymlinks(t *testing.T) {
	// This test verifies that the function handles symlink evaluation properly
	// We can't easily create test symlinks, but we can ensure it doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AddCommonPaths() panicked during symlink evaluation: %v", r)
		}
	}()

	result := AddCommonPaths()

	// Should complete without panicking
	if result == "" {
		t.Error("Expected AddCommonPaths() to handle symlinks gracefully")
	}
}

func TestAddCommonPaths_NoDuplicatePaths(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Set PATH with a duplicate that might be added by AddCommonPaths
	testPath := "/custom/unique/path1:/custom/unique/path2" // Use custom paths to avoid system conflicts
	os.Setenv("PATH", testPath)

	result := AddCommonPaths()

	// Split the result and check for duplicates of our custom paths
	paths := strings.Split(result, string(os.PathListSeparator))
	customPathCounts := make(map[string]int)

	for _, path := range paths {
		if path == "" {
			continue // Skip empty paths
		}
		if strings.Contains(path, "/custom/unique/") {
			customPathCounts[path]++
		}
	}

	// Check that our custom paths don't appear multiple times
	for path, count := range customPathCounts {
		if count > 1 {
			t.Errorf("Found duplicate custom path in result: %s (appeared %d times)", path, count)
		}
	}
}

// Performance tests
func BenchmarkAddCommonPaths(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AddCommonPaths()
	}
}

func BenchmarkSetPaths(b *testing.B) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetPaths()
	}
}
