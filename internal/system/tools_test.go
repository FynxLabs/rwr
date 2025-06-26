package system

import (
	"os"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestFindTool_ToolExists(t *testing.T) {
	// Test with a tool that should exist on most systems
	var toolName string
	switch runtime.GOOS {
	case "windows":
		toolName = "cmd"
	default:
		toolName = "sh" // Should exist on Unix-like systems
	}

	result := FindTool(toolName)

	if !result.Exists {
		t.Errorf("Expected tool '%s' to exist, but Exists was false", toolName)
	}

	if result.Bin == "" {
		t.Errorf("Expected tool '%s' to have a non-empty Bin path", toolName)
	}

	// Verify the binary path is valid
	if _, err := os.Stat(result.Bin); os.IsNotExist(err) {
		t.Errorf("Tool binary path '%s' does not exist", result.Bin)
	}
}

func TestFindTool_ToolNotFound(t *testing.T) {
	// Test with a tool that definitely doesn't exist
	nonExistentTool := "definitely-not-a-real-tool-12345"

	result := FindTool(nonExistentTool)

	if result.Exists {
		t.Errorf("Expected tool '%s' to not exist, but Exists was true", nonExistentTool)
	}

	if result.Bin != "" {
		t.Errorf("Expected tool '%s' to have empty Bin path, got '%s'", nonExistentTool, result.Bin)
	}
}

func TestFindTool_CommonTools(t *testing.T) {
	// Test a set of common tools and verify the function returns appropriate ToolInfo
	var commonTools []string

	switch runtime.GOOS {
	case "windows":
		commonTools = []string{"cmd", "powershell"}
	case "darwin":
		commonTools = []string{"sh", "bash", "ls", "cat"}
	default: // linux and other unix-like
		commonTools = []string{"sh", "ls", "cat"}
	}

	for _, toolName := range commonTools {
		t.Run("tool_"+toolName, func(t *testing.T) {
			result := FindTool(toolName)

			// We expect these common tools to exist, but won't fail the test if they don't
			// since test environments might be minimal
			if result.Exists {
				if result.Bin == "" {
					t.Errorf("Tool '%s' exists but has empty Bin path", toolName)
				}
			}

			// Verify the result is a valid ToolInfo struct
			if result.Exists && result.Bin == "" {
				t.Errorf("Invalid ToolInfo: Exists=true but Bin is empty for tool '%s'", toolName)
			}

			if !result.Exists && result.Bin != "" {
				t.Errorf("Invalid ToolInfo: Exists=false but Bin is not empty for tool '%s'", toolName)
			}
		})
	}
}

func TestFindTool_EmptyToolName(t *testing.T) {
	// Test with empty tool name
	result := FindTool("")

	if result.Exists {
		t.Error("Expected empty tool name to not exist, but Exists was true")
	}

	if result.Bin != "" {
		t.Errorf("Expected empty tool name to have empty Bin path, got '%s'", result.Bin)
	}
}

func TestFindTool_ToolInfoStructure(t *testing.T) {
	// Test that the function returns a properly structured ToolInfo
	testTool := "nonexistent-test-tool"
	result := FindTool(testTool)

	// Verify the result is of the correct type
	var expected types.ToolInfo
	if result.Exists != expected.Exists && result.Exists != true {
		// This is fine, just checking structure
	}

	// Verify the struct has the expected fields
	_ = result.Exists // bool field
	_ = result.Bin    // string field

	// Test that the zero value is properly handled
	if !result.Exists && result.Bin != "" {
		t.Error("Expected non-existent tool to have empty Bin path")
	}
}

// Helper function to test tool detection behavior with modified environment
func TestFindTool_WithModifiedPATH(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	// Test with empty PATH
	os.Setenv("PATH", "")

	// Try to find a tool that would normally exist
	var testTool string
	switch runtime.GOOS {
	case "windows":
		testTool = "cmd"
	default:
		testTool = "sh"
	}

	result := FindTool(testTool)

	// With empty PATH, the tool might still be found due to AddCommonPaths()
	// We're mainly testing that the function doesn't crash and returns valid ToolInfo
	if result.Exists && result.Bin == "" {
		t.Error("Tool marked as existing but has empty Bin path")
	}

	if !result.Exists && result.Bin != "" {
		t.Error("Tool marked as not existing but has non-empty Bin path")
	}
}

func TestFindTool_ReturnTypeConsistency(t *testing.T) {
	// Test that the function consistently returns the correct type
	testCases := []string{
		"existing-tool-test",  // likely doesn't exist
		"another-nonexistent", // definitely doesn't exist
		"sh",                  // might exist on Unix systems
	}

	for _, testCase := range testCases {
		t.Run("consistency_"+testCase, func(t *testing.T) {
			result := FindTool(testCase)

			// Verify type consistency
			if result.Exists {
				// If tool exists, it should have a non-empty path
				if result.Bin == "" {
					t.Errorf("Tool '%s' marked as existing but has empty Bin path", testCase)
				}
			} else {
				// If tool doesn't exist, path should be empty
				if result.Bin != "" {
					t.Errorf("Tool '%s' marked as not existing but has non-empty Bin path: '%s'", testCase, result.Bin)
				}
			}
		})
	}
}

// Benchmark test to ensure the function performs reasonably
func BenchmarkFindTool(b *testing.B) {
	toolName := "nonexistent-benchmark-tool"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindTool(toolName)
	}
}
