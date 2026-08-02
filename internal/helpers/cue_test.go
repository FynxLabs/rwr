package helpers

import (
	"strings"
	"testing"
)

type cueTestBlueprint struct {
	Packages []struct {
		Name    string `json:"name" yaml:"name"`
		Action  string `json:"action" yaml:"action"`
		Version string `json:"version,omitempty" yaml:"version,omitempty"`
	} `json:"packages" yaml:"packages"`
}

// A CUE blueprint decodes identically to its YAML twin: evaluation exports
// concrete JSON, which rides the same strict decode path.
func TestUnmarshalBlueprint_CUEEqualsYAMLTwin(t *testing.T) {
	cueData := []byte(`
packages: [
	{name: "git", action: "install"},
	{name: "vim", action: "install", version: "9.0"},
]
`)
	yamlData := []byte(`
packages:
  - name: git
    action: install
  - name: vim
    action: install
    version: "9.0"
`)

	var fromCUE, fromYAML cueTestBlueprint
	if err := UnmarshalBlueprint(cueData, "cue", &fromCUE); err != nil {
		t.Fatalf("UnmarshalBlueprint(cue): %v", err)
	}
	if err := UnmarshalBlueprint(yamlData, "yaml", &fromYAML); err != nil {
		t.Fatalf("UnmarshalBlueprint(yaml): %v", err)
	}

	if len(fromCUE.Packages) != len(fromYAML.Packages) {
		t.Fatalf("CUE decoded %d packages, YAML %d", len(fromCUE.Packages), len(fromYAML.Packages))
	}
	for i := range fromYAML.Packages {
		if fromCUE.Packages[i] != fromYAML.Packages[i] {
			t.Errorf("package %d differs: cue=%+v yaml=%+v", i, fromCUE.Packages[i], fromYAML.Packages[i])
		}
	}
}

// CUE's value: constraints hold at decode time — a violated constraint is an
// error naming the position, before anything touches the machine.
func TestUnmarshalBlueprint_CUEConstraintViolation(t *testing.T) {
	cueData := []byte(`
#Package: {name: string, action: "install" | "remove"}
packages: [...#Package]
packages: [{name: "git", action: "explode"}]
`)
	var out cueTestBlueprint
	err := UnmarshalBlueprint(cueData, "cue", &out)
	if err == nil {
		t.Fatal("constraint violation decoded successfully, want an error")
	}
	if !strings.Contains(err.Error(), "action") {
		t.Errorf("error %q does not name the failing field", err)
	}
}

// A non-concrete value (an unresolved field) is an authoring error, not a
// value to guess at.
func TestUnmarshalBlueprint_CUENonConcreteIsError(t *testing.T) {
	cueData := []byte(`packages: [{name: "git", action: "install", version: string}]`)
	var out cueTestBlueprint
	err := UnmarshalBlueprint(cueData, "cue", &out)
	if err == nil {
		t.Fatal("non-concrete value decoded successfully, want an error")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error %q does not name the unresolved field", err)
	}
}

// Evaluation is sandboxed: with no module loader and no filesystem root, a
// .cue file can import only the built-in standard library — a package path
// that would resolve through disk or a registry fails to compile.
func TestEvalCUE_ImportsOutsideStdlibRefused(t *testing.T) {
	for name, src := range map[string]string{
		"module path":   `import "example.com/evil"` + "\n" + `packages: evil.list`,
		"relative path": `import "../../../etc/secrets"` + "\n" + `packages: secrets.list`,
	} {
		if _, err := EvalCUEToJSON([]byte(src), "test.cue"); err == nil {
			t.Errorf("%s: evaluation succeeded, want a refusal", name)
		}
	}
}

// The built-in stdlib stays available: pure computation, no I/O.
func TestEvalCUE_StdlibWorks(t *testing.T) {
	src := `
import "strings"

packages: [{name: strings.ToLower("GIT"), action: "install"}]
`
	out, err := EvalCUEToJSON([]byte(src), "test.cue")
	if err != nil {
		t.Fatalf("EvalCUEToJSON: %v", err)
	}
	if !strings.Contains(string(out), `"git"`) {
		t.Errorf("output %s does not contain the computed value", out)
	}
}
