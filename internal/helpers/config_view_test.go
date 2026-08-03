package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	viper.Set("rwr.configdir", dir)
	t.Cleanup(func() { viper.Set("rwr.configdir", "") })
	return dir
}

// Secrets never reach the terminal by default; --show-secrets is the explicit
// opt-in.
func TestConfigView_RedactsSecrets(t *testing.T) {
	withConfigDir(t)
	viper.Set("repository.gh_api_token", "ghp_supersecret")
	defer viper.Set("repository.gh_api_token", "")

	out, err := ConfigView(false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "ghp_supersecret") {
		t.Fatalf("secret leaked:\n%s", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatalf("redaction placeholder missing:\n%s", out)
	}

	shown, err := ConfigView(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown, "ghp_supersecret") {
		t.Fatalf("--show-secrets did not reveal the value:\n%s", shown)
	}
}

// A key set in the config file is annotated as coming from it.
func TestConfigView_AnnotatesConfigFileKeys(t *testing.T) {
	dir := withConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("log:\n  level: debug\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := ConfigView(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "log.level") {
			if !strings.Contains(line, "(config)") {
				t.Fatalf("log.level not annotated as config: %s", line)
			}
			return
		}
	}
	t.Fatalf("log.level missing from view:\n%s", out)
}

// EditConfig runs the operator's $EDITOR against the config path and creates
// a default file first when none exists.
func TestEditConfig_InvokesEditorAndCreates(t *testing.T) {
	dir := withConfigDir(t)
	record := filepath.Join(dir, "invoked")
	editor := filepath.Join(dir, "fake-editor.sh")
	if err := os.WriteFile(editor, []byte("#!/bin/sh\necho \"$1\" > "+record+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)

	if err := EditConfig(); err != nil {
		t.Fatal(err)
	}

	invoked, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("editor never invoked: %v", err)
	}
	wantPath := filepath.Join(dir, "config.yaml")
	if strings.TrimSpace(string(invoked)) != wantPath {
		t.Fatalf("editor got %q, want %q", strings.TrimSpace(string(invoked)), wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("config file not created before editing: %v", err)
	}
}
