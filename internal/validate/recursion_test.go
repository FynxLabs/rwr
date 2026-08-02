package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// Blueprints live in subdirectories — packages/, files/, services/ — which is the
// layout the documentation recommends and every shipped example uses. Validation
// read only the top directory, so `rwr validate` on a real tree checked the init
// file and reported success. These tests use that layout.

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestValidateBlueprints_FindsProblemsInSubdirectories(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml": "blueprints:\n  format: yaml\n",
		// action "explode" is not a package action; nothing but recursion finds it.
		"packages/dev.yaml": "packages:\n  - name: git\n    action: explode\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatalf("validation returned an error: %v", err)
	}

	var found bool
	for _, issue := range results.Issues {
		if strings.Contains(issue.Message, "explode") {
			found = true
		}
	}
	if !found {
		t.Fatalf("invalid action in packages/dev.yaml was not reported; issues were: %+v", results.Issues)
	}
}

func TestValidateBlueprints_AcceptsAValidNestedTree(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml":             "blueprints:\n  format: yaml\n",
		"packages/base.yaml":    "packages:\n  - name: git\n    action: install\n",
		"packages/many.yaml":    "packages:\n  - names: [curl, wget]\n    action: install\n",
		"services/base.yaml":    "services:\n  - name: sshd\n    action: enable\n",
		"files/dotfiles.yaml":   "files:\n  - source: ./src/vimrc\n    target: /tmp/vimrc\n    action: copy\n",
		"scripts/setup.yaml":    "scripts:\n  - name: setup.sh\n    action: run\n    source: ./bin\n",
		"git/repos.yaml":        "git:\n  - name: dots\n    action: clone\n    url: https://example.invalid/d.git\n    path: /tmp/dots\n",
		"fonts/nerd.yaml":       "fonts:\n  - name: FiraCode\n    action: install\n",
		"configuration/ui.yaml": "configurations:\n  - name: theme\n    tool: dconf\n    action: set\n    key: /org/gnome/theme\n    value: dark\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatalf("validation returned an error: %v", err)
	}

	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			t.Errorf("valid tree reported an error: %s [%s]", issue.Message, issue.File)
		}
	}
}

// A package entry names one package with `name` or several with `names`. Both
// forms are used throughout the examples and both must validate.
func TestValidatePackages_AcceptsNameOrNames(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml":          "blueprints:\n  format: yaml\n",
		"packages/one.yaml":  "packages:\n  - name: git\n    action: install\n",
		"packages/many.yaml": "packages:\n  - names: [curl, wget]\n    action: install\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatal(err)
	}
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			t.Errorf("unexpected error: %s [%s]", issue.Message, issue.File)
		}
	}
}

// An entry with neither is genuinely wrong and must still be caught.
func TestValidatePackages_RejectsEntryWithNoName(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml":            "blueprints:\n  format: yaml\n",
		"packages/broken.yaml": "packages:\n  - action: install\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError && strings.Contains(issue.Message, "name") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a package with neither name nor names was accepted; issues: %+v", results.Issues)
	}
}

// Blueprints are rendered as templates before they are read. Validating the raw
// bytes reports a parse error against a blueprint that works, because {{ }} is a
// flow mapping in YAML.
func TestValidateBlueprints_RendersTemplates(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml": "blueprints:\n  format: yaml\n",
		"files/dotfiles.yaml": "files:\n  - source: ./src/vimrc\n" +
			"    target: \"{{ .User.home }}/.vimrc\"\n    action: copy\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatal(err)
	}
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			t.Errorf("templated blueprint reported an error: %s", issue.Message)
		}
	}
}

// A schema version this build cannot read must be refused by validation too, or
// `rwr validate` says a tree is sound that a run will refuse.
func TestValidateBlueprints_RejectsUnsupportedSchemaVersion(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml":         "blueprints:\n  format: yaml\n",
		"packages/new.yaml": "schema_version: 99\npackages:\n  - name: git\n    action: install\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, issue := range results.Issues {
		if strings.Contains(issue.Message, "99") {
			found = true
		}
	}
	if !found {
		t.Fatalf("unsupported schema version was not reported; issues: %+v", results.Issues)
	}
}

// .git holds a great many files and none of them are blueprints.
func TestValidateBlueprints_SkipsDotDirectories(t *testing.T) {
	root := writeTree(t, map[string]string{
		"init.yaml":            "blueprints:\n  format: yaml\n",
		".git/packages/x.yaml": "packages:\n  - action: explode\n",
	})

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(root, false, results, nil); err != nil {
		t.Fatal(err)
	}
	for _, issue := range results.Issues {
		if strings.Contains(issue.File, ".git") {
			t.Errorf("validated a file under .git: %s", issue.File)
		}
	}
}
