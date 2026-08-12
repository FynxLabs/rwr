package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// Imported blueprint files resolve templates against the run's variables:
// without this, `{{ .User.home }}` in an imported file survived to execution
// as a literal path and rwr created a directory named "{{ .User.home }}".
func TestResolveImports_ResolvesTemplatesInImports(t *testing.T) {
	dir := t.TempDir()
	imported := filepath.Join(dir, "shared.yaml")
	if err := os.WriteFile(imported, []byte("git:\n  - name: rwr\n    path: \"{{ .User.home }}/git/rwr\"\n    action: clone\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	vars := types.Variables{}
	vars.User.Home = "/home/test"
	defer SetTemplateVariables(&vars)()

	type entry struct {
		Name   string `yaml:"name"`
		Path   string `yaml:"path"`
		Import string `yaml:"import"`
	}
	items := []entry{{Import: "shared.yaml"}}
	resolved, err := ResolveImports(items, dir,
		func(e entry) string { return e.Import },
		func(data []byte, format string) ([]entry, error) {
			var d struct {
				Git []entry `yaml:"git"`
			}
			if err := UnmarshalBlueprint(data, format, &d); err != nil {
				return nil, err
			}
			return d.Git, nil
		}, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("got %d items, want 1", len(resolved))
	}
	if resolved[0].Path != "/home/test/git/rwr" {
		t.Fatalf("imported path not template-resolved: %q", resolved[0].Path)
	}
}
