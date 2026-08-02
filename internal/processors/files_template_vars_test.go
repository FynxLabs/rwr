package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// A template's own `variables` are overrides for that one render. They were
// written into initConfig's shared UserDefined map, so template A's override
// leaked into template B and permanently clobbered a run-wide variable of the
// same name.
func TestProcessFiles_TemplateVariablesDoNotLeak(t *testing.T) {
	tempDir := t.TempDir()
	blueprintDir := filepath.Join(tempDir, "blueprints")
	if err := os.MkdirAll(filepath.Join(blueprintDir, "tpl"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Both templates render the same variable.
	for _, name := range []string{"a.conf", "b.conf"} {
		if err := os.WriteFile(filepath.Join(blueprintDir, "tpl", name), []byte("greeting={{ .UserDefined.greeting }}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	config := &types.InitConfig{
		Init: types.Init{Location: blueprintDir, Format: "yaml"},
		Variables: types.Variables{
			Flags:       types.Flags{Debug: true},
			UserDefined: map[string]interface{}{"greeting": "global"},
		},
	}

	// a.conf overrides greeting; b.conf must still see the run-wide value.
	blueprintData := `
templates:
  - name: "a.conf"
    action: "create"
    source: "tpl"
    target: "` + tempDir + `/"
    variables:
      greeting: "local"
  - name: "b.conf"
    action: "create"
    source: "tpl"
    target: "` + tempDir + `/"
`

	if err := ProcessFiles([]byte(blueprintData), blueprintDir, "yaml", &types.OSInfo{}, config); err != nil {
		t.Fatalf("ProcessFiles: %v", err)
	}

	read := func(name string) string {
		t.Helper()
		got, err := os.ReadFile(filepath.Join(tempDir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		return string(got)
	}

	if got := read("a.conf"); !strings.Contains(got, "greeting=local") {
		t.Errorf("a.conf = %q, want its own override %q", got, "local")
	}
	if got := read("b.conf"); !strings.Contains(got, "greeting=global") {
		t.Errorf("b.conf = %q, want the run-wide value %q — a.conf's override leaked", got, "global")
	}
	if got := config.Variables.UserDefined["greeting"]; got != "global" {
		t.Errorf("initConfig.Variables.UserDefined[greeting] = %v, want %q untouched", got, "global")
	}
}
