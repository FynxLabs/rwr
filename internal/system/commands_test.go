package system

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestCommandExists_ValidCommand(t *testing.T) {
	// Test with a command that should exist on most systems
	var testCommand string
	switch runtime.GOOS {
	case "windows":
		testCommand = "cmd"
	default:
		testCommand = "sh"
	}

	result := CommandExists(testCommand)

	if !result {
		t.Errorf("Expected CommandExists('%s') to be true, got false", testCommand)
	}
}

func TestCommandExists_InvalidCommand(t *testing.T) {
	// Test with a command that definitely doesn't exist
	invalidCommand := "definitely-not-a-real-command-12345"

	result := CommandExists(invalidCommand)

	if result {
		t.Errorf("Expected CommandExists('%s') to be false, got true", invalidCommand)
	}
}

func TestCommandExists_EmptyCommand(t *testing.T) {
	result := CommandExists("")

	if result {
		t.Error("Expected CommandExists('') to be false, got true")
	}
}

func TestGetBinPath_ValidCommand(t *testing.T) {
	// Test with a command that should exist
	var testCommand string
	switch runtime.GOOS {
	case "windows":
		testCommand = "cmd"
	default:
		testCommand = "sh"
	}

	path, err := GetBinPath(testCommand)

	if err != nil {
		t.Errorf("Expected no error for GetBinPath('%s'), got: %v", testCommand, err)
	}

	if path == "" {
		t.Errorf("Expected non-empty path for GetBinPath('%s'), got empty string", testCommand)
	}

	// Verify the path is clean (no redundant separators)
	if filepath.Clean(path) != path {
		t.Errorf("Expected clean path, got potentially unclean path: %s", path)
	}
}

func TestGetBinPath_InvalidCommand(t *testing.T) {
	invalidCommand := "definitely-not-a-real-command-12345"

	path, err := GetBinPath(invalidCommand)

	if err == nil {
		t.Errorf("Expected error for GetBinPath('%s'), got nil", invalidCommand)
	}

	if path != "" {
		t.Errorf("Expected empty path for invalid command, got: %s", path)
	}
}

func TestGetBinPath_EmptyCommand(t *testing.T) {
	path, err := GetBinPath("")

	if err == nil {
		t.Error("Expected error for GetBinPath(''), got nil")
	}

	if path != "" {
		t.Errorf("Expected empty path for empty command, got: %s", path)
	}
}

// Test Command struct validation and basic functionality
func TestCommand_BasicStructure(t *testing.T) {
	cmd := types.Command{
		Exec:        "echo",
		Args:        []string{"hello", "world"},
		Variables:   map[string]string{"TEST_VAR": "test_value"},
		LogName:     "test.log",
		AsUser:      "testuser",
		Interactive: false,
		Elevated:    false,
	}

	// Test that struct fields are properly set
	if cmd.Exec != "echo" {
		t.Errorf("Expected Exec to be 'echo', got '%s'", cmd.Exec)
	}

	if len(cmd.Args) != 2 {
		t.Errorf("Expected Args length to be 2, got %d", len(cmd.Args))
	}

	if cmd.Args[0] != "hello" || cmd.Args[1] != "world" {
		t.Errorf("Expected Args to be ['hello', 'world'], got %v", cmd.Args)
	}

	if cmd.Variables["TEST_VAR"] != "test_value" {
		t.Errorf("Expected Variables['TEST_VAR'] to be 'test_value', got '%s'", cmd.Variables["TEST_VAR"])
	}

	if cmd.LogName != "test.log" {
		t.Errorf("Expected LogName to be 'test.log', got '%s'", cmd.LogName)
	}

	if cmd.AsUser != "testuser" {
		t.Errorf("Expected AsUser to be 'testuser', got '%s'", cmd.AsUser)
	}

	if cmd.Interactive != false {
		t.Errorf("Expected Interactive to be false, got %v", cmd.Interactive)
	}

	if cmd.Elevated != false {
		t.Errorf("Expected Elevated to be false, got %v", cmd.Elevated)
	}
}

func TestCommand_EmptyCommand(t *testing.T) {
	cmd := types.Command{}

	// Test zero values
	if cmd.Exec != "" {
		t.Errorf("Expected empty Exec, got '%s'", cmd.Exec)
	}

	if len(cmd.Args) != 0 {
		t.Errorf("Expected empty Args, got %v", cmd.Args)
	}

	if len(cmd.Variables) != 0 {
		t.Errorf("Expected empty Variables, got %v", cmd.Variables)
	}

	if cmd.LogName != "" {
		t.Errorf("Expected empty LogName, got '%s'", cmd.LogName)
	}

	if cmd.AsUser != "" {
		t.Errorf("Expected empty AsUser, got '%s'", cmd.AsUser)
	}

	if cmd.Interactive != false {
		t.Errorf("Expected Interactive to be false, got %v", cmd.Interactive)
	}

	if cmd.Elevated != false {
		t.Errorf("Expected Elevated to be false, got %v", cmd.Elevated)
	}
}

func TestCommand_WithNilVariables(t *testing.T) {
	cmd := types.Command{
		Variables: nil, // Explicitly set to nil
	}
	_ = cmd.Exec // Assign values to avoid unused write warnings
	_ = cmd.Args

	// Should handle nil Variables gracefully
	if cmd.Variables != nil {
		t.Errorf("Expected Variables to be nil, got %v", cmd.Variables)
	}

	// Should be safe to check length of nil map
	if len(cmd.Variables) != 0 {
		t.Errorf("Expected Variables length to be 0, got %d", len(cmd.Variables))
	}
}

func TestCommand_WithEmptyArgs(t *testing.T) {
	cmd := types.Command{
		Args: []string{}, // Empty args
	}
	_ = cmd.Exec // Assign value to avoid unused write warning

	if len(cmd.Args) != 0 {
		t.Errorf("Expected Args length to be 0, got %d", len(cmd.Args))
	}
}

func TestCommand_WithSpecialCharacters(t *testing.T) {
	cmd := types.Command{
		Args: []string{"hello world", "special!@#$%^&*()", "unicode-ñáéíóú"},
		Variables: map[string]string{
			"SPECIAL_VAR": "value with spaces",
			"UNICODE_VAR": "ñáéíóú",
			"SYMBOLS_VAR": "!@#$%^&*()",
		},
	}
	_ = cmd.Exec // Assign value to avoid unused write warning

	// Test that special characters are preserved
	if cmd.Args[0] != "hello world" {
		t.Errorf("Expected Args[0] to be 'hello world', got '%s'", cmd.Args[0])
	}

	if cmd.Args[1] != "special!@#$%^&*()" {
		t.Errorf("Expected Args[1] to contain special chars, got '%s'", cmd.Args[1])
	}

	if cmd.Variables["SPECIAL_VAR"] != "value with spaces" {
		t.Errorf("Expected Variables to handle spaces, got '%s'", cmd.Variables["SPECIAL_VAR"])
	}

	if cmd.Variables["UNICODE_VAR"] != "ñáéíóú" {
		t.Errorf("Expected Variables to handle unicode, got '%s'", cmd.Variables["UNICODE_VAR"])
	}
}

// Mock tests for command execution - these test the structure without actually running commands
func TestCommand_ExecutionFlags(t *testing.T) {
	testCases := []struct {
		name        string
		cmd         types.Command
		expectedCmd string
	}{
		{
			name: "Basic command",
			cmd: types.Command{
				Exec: "echo",
				Args: []string{"hello"},
			},
			expectedCmd: "echo",
		},
		{
			name: "Elevated command",
			cmd: types.Command{
				Exec:     "echo",
				Args:     []string{"hello"},
				Elevated: true,
			},
			expectedCmd: "echo",
		},
		{
			name: "User command",
			cmd: types.Command{
				Exec:   "echo",
				Args:   []string{"hello"},
				AsUser: "testuser",
			},
			expectedCmd: "echo",
		},
		{
			name: "Interactive command",
			cmd: types.Command{
				Exec:        "echo",
				Args:        []string{"hello"},
				Interactive: true,
			},
			expectedCmd: "echo",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cmd.Exec != tc.expectedCmd {
				t.Errorf("Expected Exec to be '%s', got '%s'", tc.expectedCmd, tc.cmd.Exec)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCommandExists(b *testing.B) {
	command := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CommandExists(command)
	}
}

func BenchmarkGetBinPath(b *testing.B) {
	command := "sh"
	if runtime.GOOS == "windows" {
		command = "cmd"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetBinPath(command)
	}
}
