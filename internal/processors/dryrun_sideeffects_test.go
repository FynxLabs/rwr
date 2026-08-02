package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// gitBlueprintConfig builds an init config whose blueprint source is a git repo
// targeting target.
func gitBlueprintConfig(target string) *types.InitConfig {
	cfg := &types.InitConfig{}
	cfg.Init.Git = &types.GitOptions{
		URL:    "https://example.invalid/blueprints.git",
		Target: target,
	}
	return cfg
}

func TestGetBlueprints_DryRunLeavesNonGitTargetIntact(t *testing.T) {
	target := filepath.Join(t.TempDir(), "dotfiles")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	marker := filepath.Join(target, "precious.txt")
	if err := os.WriteFile(marker, []byte("do not delete"), 0o600); err != nil {
		t.Fatalf("failed to seed target: %v", err)
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)

	got, err := GetBlueprints(gitBlueprintConfig(target))
	if err != nil {
		t.Fatalf("GetBlueprints returned an error in dry-run: %v", err)
	}
	if got != target {
		t.Errorf("expected blueprint location %q, got %q", target, got)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("dry-run removed contents of the blueprint target: %v", err)
	}
}

func TestGetBlueprints_NonGitTargetErrorsInsteadOfDeleting(t *testing.T) {
	target := filepath.Join(t.TempDir(), "dotfiles")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}
	marker := filepath.Join(target, "precious.txt")
	if err := os.WriteFile(marker, []byte("do not delete"), 0o600); err != nil {
		t.Fatalf("failed to seed target: %v", err)
	}

	system.SetDryRun(false)

	if _, err := GetBlueprints(gitBlueprintConfig(target)); err == nil {
		t.Fatal("expected an error for a non-git blueprint target, got nil")
	} else if !containsString(err.Error(), "not a git repository") {
		t.Errorf("expected an actionable 'not a git repository' error, got: %v", err)
	}

	if _, err := os.Stat(marker); err != nil {
		t.Errorf("blueprint target was deleted instead of reported: %v", err)
	}
}

func TestWriteBootstrapMarker_SkippedInDryRun(t *testing.T) {
	configDir := t.TempDir()
	viper.Set("rwr.configdir", configDir)
	defer viper.Set("rwr.configdir", "")

	marker := filepath.Join(configDir, "bootstrap")

	system.SetDryRun(true)
	if err := writeBootstrapMarker(); err != nil {
		t.Fatalf("writeBootstrapMarker returned an error in dry-run: %v", err)
	}
	system.SetDryRun(false)

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote the bootstrap marker at %s (stat err: %v)", marker, err)
	}

	if err := writeBootstrapMarker(); err != nil {
		t.Fatalf("writeBootstrapMarker returned an error: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("expected the bootstrap marker to be written outside dry-run: %v", err)
	}
}
