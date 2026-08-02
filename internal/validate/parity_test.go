package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// The stage-1 resolve refactor (add-tui task 1) must not change what validate
// reports. This pins the current output shape on a fixture tree carrying one
// of each diagnostic class; it was written against the pre-refactor code and
// has to stay green after ValidateBlueprints consumes the Plan.
func TestValidate_Stage1Parity(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("init.yaml", "blueprints:\n  format: yaml\n  location: "+dir+"\n")
	// Clean file: no diagnostics.
	write("packages/good.yaml", "packages:\n  - name: git\n    action: install\n")
	// Unknown key: strict decode error.
	write("packages/unknown-key.yaml", "packages:\n  - name: vim\n    action: install\n    banana: true\n")
	// Fixed-namespace template typo: validation error.
	write("files/typo.yaml", "files:\n  - name: rc\n    action: create\n    target: \"{{ .User.hoem }}/.rc\"\n    content: x\n")
	// Missing required field: component-validator error.
	write("services/incomplete.yaml", "services:\n  - action: enable\n")

	results := &types.ValidationResults{}
	if err := ValidateBlueprints(dir, false, results, &types.OSInfo{}); err != nil {
		t.Fatalf("ValidateBlueprints: %v", err)
	}

	assertIssue := func(fragment string) {
		t.Helper()
		for _, issue := range results.Issues {
			if strings.Contains(issue.Message, fragment) {
				return
			}
		}
		t.Errorf("no issue contains %q; issues: %+v", fragment, results.Issues)
	}

	assertIssue("banana")     // unknown key named
	assertIssue(".User.hoem") // template typo named
	assertIssue("name")       // missing service name

	for _, issue := range results.Issues {
		if strings.Contains(issue.File, "good.yaml") {
			t.Errorf("clean file produced an issue: %+v", issue)
		}
	}
	errors := 0
	for _, issue := range results.Issues {
		if issue.Severity == types.ValidationError {
			errors++
		}
	}
	if errors < 3 {
		t.Errorf("error issues = %d, want >= 3", errors)
	}
}
