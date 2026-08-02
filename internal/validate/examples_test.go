package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// examplesDir locates the repository's examples/ tree from this package.
func examplesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "examples")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("examples directory not found: %v", err)
	}
	return dir
}

func walkExamples(t *testing.T, fn func(path, ext string, data []byte)) {
	t.Helper()
	root := examplesDir(t)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		switch ext {
		case "yaml", "yml", "json", "toml", "cue":
		default:
			return nil
		}
		data, readErr := os.ReadFile(path) // #nosec G304 -- test walks the repo's own examples tree
		if readErr != nil {
			t.Errorf("%s: %v", path, readErr)
			return nil
		}
		fn(path, ext, data)
		return nil
	})
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
}

// Every shipped example must parse in the format its extension claims.
//
// Nineteen of them did not: fifteen YAML files started a value with an unquoted
// `{{`, which YAML reads as a flow mapping, and four files with a .json extension
// contained YAML. Nothing caught it because no test ever read the examples.
func TestExamples_ParseInTheirDeclaredFormat(t *testing.T) {
	walkExamples(t, func(path, ext string, data []byte) {
		// Templates are resolved before parsing at runtime, so do the same here;
		// a file is allowed to contain template syntax, not to be unparseable.
		resolved, err := helpers.ResolveTemplate(data, exampleVariables())
		if err != nil {
			t.Errorf("%s: template does not parse: %v", path, err)
			return
		}

		var out map[string]interface{}
		if err := helpers.UnmarshalBlueprint(resolved, ext, &out); err != nil {
			t.Errorf("%s: does not parse as %s: %v", path, ext, err)
		}
	})
}

// The examples may only use template variables that actually exist. A missing key
// renders as the literal "<no value>" rather than erroring, so a typo silently
// produces paths like "<no value>/.bashrc" — which is how 214 uses of
// {{ .User.Home }} (capital H, where the map key is "home") went unnoticed.
func TestExamples_UseOnlyRealTemplateVariables(t *testing.T) {
	walkExamples(t, func(path, ext string, data []byte) {
		resolved, err := helpers.ResolveTemplate(data, exampleVariables())
		if err != nil {
			return // reported by the parse test
		}
		if strings.Contains(string(resolved), "<no value>") {
			t.Errorf("%s: renders \"<no value>\" — it references a template variable "+
				"that does not exist (check casing: the keys are lowercase-first)", path)
		}
	})
}

// exampleVariables mirrors what Initialize builds, with every user-defined value
// the examples reference. A template referring to something absent renders
// "<no value>", which the test above rejects.
func exampleVariables() types.Variables {
	return types.Variables{
		User: types.UserInfo{
			Username:  "example",
			FirstName: "Example",
			LastName:  "User",
			FullName:  "Example User",
			GroupName: "example",
			Home:      "/home/example",
			Shell:     "/bin/bash",
		},
		System: types.System{
			OS:        "linux",
			OSFamily:  "arch",
			OSVersion: "rolling",
			OSArch:    "amd64",
		},
		UserDefined: map[string]interface{}{
			"project_name":   "example-project",
			"repo_url":       "https://github.com/example/dotfiles.git",
			"company":        "MyCompany",
			"department":     "Engineering",
			"editor_theme":   "dracula",
			"editor_plugins": "basic",
		},
	}
}

// A CUE blueprint that violates its own constraints is a validate diagnostic
// with the file attached, on the same surface as schema errors — and the tree
// still gets its init file discovered in .cue.
func TestValidate_CueConstraintViolationIsDiagnostic(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("init.cue", `init: {format: "cue", location: "`+dir+`"}`)
	writeFile("packages/bad.cue", `
#Package: {name: string, action: "install" | "remove"}
packages: [...#Package]
packages: [{name: "git", action: "explode"}]
`)

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(dir, false, results, &types.OSInfo{}); err != nil {
		t.Fatalf("ValidateBlueprints: %v", err)
	}
	found := false
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError && strings.Contains(issue.Message, "action") {
			found = true
			if issue.File == "" {
				t.Error("diagnostic carries no file")
			}
		}
	}
	if !found {
		t.Errorf("no diagnostic names the failed constraint; issues: %+v", results.Issues)
	}
}
