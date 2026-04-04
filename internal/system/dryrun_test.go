package system

import (
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestSetDryRun_EnablesMode(t *testing.T) {
	// Ensure clean state
	SetDryRun(false)

	SetDryRun(true)
	if !IsDryRun() {
		t.Error("Expected IsDryRun() to return true after SetDryRun(true)")
	}

	SetDryRun(false)
	if IsDryRun() {
		t.Error("Expected IsDryRun() to return false after SetDryRun(false)")
	}
}

func TestSetDryRun_DefaultOff(t *testing.T) {
	// Reset to default
	dryRunMode = false
	if IsDryRun() {
		t.Error("Expected dry-run mode to be off by default")
	}
}

func TestRunCommand_DryRun_SkipsExecution(t *testing.T) {
	SetDryRun(true)
	defer SetDryRun(false)

	// This command would fail if actually executed (no such binary)
	cmd := types.Command{
		Exec: "this-binary-does-not-exist-at-all",
		Args: []string{"--fake-flag"},
	}

	err := RunCommand(cmd, false)
	if err != nil {
		t.Errorf("Expected RunCommand to return nil in dry-run mode, got: %v", err)
	}
}

func TestRunCommand_DryRun_ReturnsNilForElevated(t *testing.T) {
	SetDryRun(true)
	defer SetDryRun(false)

	cmd := types.Command{
		Exec:     "rm",
		Args:     []string{"-rf", "/"},
		Elevated: true,
	}

	err := RunCommand(cmd, false)
	if err != nil {
		t.Errorf("Expected RunCommand to return nil in dry-run mode for elevated command, got: %v", err)
	}
}

func TestRunCommandOutput_DryRun_SkipsExecution(t *testing.T) {
	SetDryRun(true)
	defer SetDryRun(false)

	cmd := types.Command{
		Exec: "this-binary-does-not-exist-at-all",
		Args: []string{"--fake-flag"},
	}

	output, err := RunCommandOutput(cmd, false)
	if err != nil {
		t.Errorf("Expected RunCommandOutput to return nil error in dry-run mode, got: %v", err)
	}
	if output != "" {
		t.Errorf("Expected RunCommandOutput to return empty string in dry-run mode, got: %q", output)
	}
}

func TestRunCommand_NoDryRun_ActuallyExecutes(t *testing.T) {
	SetDryRun(false)

	// echo should succeed on any unix system
	cmd := types.Command{
		Exec: "echo",
		Args: []string{"test"},
	}

	err := RunCommand(cmd, false)
	if err != nil {
		t.Errorf("Expected RunCommand to succeed for 'echo test', got: %v", err)
	}
}

func TestRunCommand_NoDryRun_FailsForBadCommand(t *testing.T) {
	SetDryRun(false)

	cmd := types.Command{
		Exec: "this-binary-does-not-exist-at-all",
		Args: []string{"--fake"},
	}

	err := RunCommand(cmd, false)
	if err == nil {
		t.Error("Expected RunCommand to return error for nonexistent command when dry-run is off")
	}
}

func TestRunCommandOutput_NoDryRun_ReturnsOutput(t *testing.T) {
	SetDryRun(false)

	cmd := types.Command{
		Exec: "echo",
		Args: []string{"hello"},
	}

	output, err := RunCommandOutput(cmd, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if output == "" {
		t.Error("Expected non-empty output from 'echo hello'")
	}
}
