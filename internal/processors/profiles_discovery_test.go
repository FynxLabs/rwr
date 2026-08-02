package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// `rwr profiles` used to read only the arrays written inline in the init file.
// Profiles are declared on blueprint entries, so the command that exists to tell
// an operator what --profile accepts reported "No profiles found" for every tree
// that actually uses profiles. These tests build a tree the normal way.

func writeBlueprintTree(t *testing.T, files map[string]string) string {
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

func treeConfig(root string) *types.InitConfig {
	init := &types.InitConfig{}
	init.Init.Location = root
	init.Init.Format = types.FormatYAML
	init.Variables.UserDefined = map[string]interface{}{}
	return init
}

func TestCollectProfiles_ReadsTheBlueprintTree(t *testing.T) {
	root := writeBlueprintTree(t, map[string]string{
		"packages/work.yaml": "packages:\n" +
			"  - name: slack\n    action: install\n    profiles: [work]\n" +
			"  - name: git\n    action: install\n",
		"packages/home.yaml": "packages:\n" +
			"  - name: steam\n    action: install\n    profiles: [personal, gaming]\n",
		"services/base.yaml": "services:\n" +
			"  - name: sshd\n    action: enable\n    profiles: [work]\n",
	})

	summary, err := CollectProfiles(treeConfig(root))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int{"work": 2, "personal": 1, "gaming": 1}
	if len(summary.Names) != len(want) {
		t.Fatalf("found profiles %v, want %v", summary.Names, want)
	}
	for name, count := range want {
		if summary.Counts[name] != count {
			t.Errorf("profile %q counted %d items, want %d", name, summary.Counts[name], count)
		}
	}
	if summary.BaseItems != 1 {
		t.Errorf("base items = %d, want 1", summary.BaseItems)
	}
}

func TestCollectProfiles_CoversEveryBlueprintType(t *testing.T) {
	root := writeBlueprintTree(t, map[string]string{
		"packages/p.yaml":     "packages:\n  - name: git\n    action: install\n    profiles: [a]\n",
		"repositories/r.yaml": "repositories:\n  - name: r\n    package_manager: apt\n    action: add\n    profiles: [b]\n",
		"files/f.yaml":        "files:\n  - source: ./x\n    target: /tmp/x\n    action: copy\n    profiles: [c]\n",
		"services/s.yaml":     "services:\n  - name: sshd\n    action: enable\n    profiles: [d]\n",
		"git/g.yaml":          "git:\n  - name: d\n    action: clone\n    url: https://example.invalid/d.git\n    path: /tmp/d\n    profiles: [e]\n",
		"scripts/sc.yaml":     "scripts:\n  - name: s.sh\n    action: run\n    source: ./bin\n    profiles: [f]\n",
		"users/u.yaml":        "users:\n  - name: tester\n    action: create\n    profiles: [g]\n",
	})

	summary, err := CollectProfiles(treeConfig(root))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		if summary.Counts[want] == 0 {
			t.Errorf("profile %q from its blueprint type was not found; got %v", want, summary.Names)
		}
	}
}

// The names come back sorted so the output is stable between runs.
func TestCollectProfiles_NamesAreSorted(t *testing.T) {
	root := writeBlueprintTree(t, map[string]string{
		"packages/p.yaml": "packages:\n" +
			"  - name: a\n    action: install\n    profiles: [zeta, alpha, mid]\n",
	})

	summary, err := CollectProfiles(treeConfig(root))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "mid", "zeta"}
	for i, name := range want {
		if summary.Names[i] != name {
			t.Fatalf("names = %v, want %v", summary.Names, want)
		}
	}
}

// A blueprint referencing a variable has to render before it can be read.
func TestCollectProfiles_RendersTemplates(t *testing.T) {
	root := writeBlueprintTree(t, map[string]string{
		"files/f.yaml": "files:\n  - source: ./x\n    target: \"{{ .User.home }}/x\"\n" +
			"    action: copy\n    profiles: [dots]\n",
	})

	init := treeConfig(root)
	init.Variables.User = types.UserInfo{Home: "/home/tester"}

	summary, err := CollectProfiles(init)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Counts["dots"] != 1 {
		t.Fatalf("templated blueprint contributed no profile; got %v", summary.Names)
	}
}

func TestCollectProfiles_EmptyTreeReportsNothing(t *testing.T) {
	summary, err := CollectProfiles(treeConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Names) != 0 || summary.BaseItems != 0 {
		t.Fatalf("empty tree reported %v / %d base items", summary.Names, summary.BaseItems)
	}
}

// Format is derived per file: a tree mixing formats used to be visible only to
// the half of the code that derived per file, while profile discovery and the
// run-order walk filtered on the tree-wide Init.Format and skipped the rest.
func TestCollectProfiles_MixedFormatTree(t *testing.T) {
	root := writeBlueprintTree(t, map[string]string{
		"packages/base.yaml": "packages:\n  - name: git\n    action: install\n    profiles: [dev]\n",
		"files/conf.toml":    "[[files]]\nname = \"conf\"\naction = \"create\"\ntarget = \"/tmp/conf\"\ncontent = \"x\"\nprofiles = [\"work\"]\n",
	})

	summary, err := CollectProfiles(treeConfig(root))
	if err != nil {
		t.Fatalf("CollectProfiles: %v", err)
	}
	if summary.Files != 2 {
		t.Errorf("Files = %d, want 2 (both formats seen)", summary.Files)
	}
	for _, want := range []string{"dev", "work"} {
		found := false
		for _, name := range summary.Names {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("profile %q not discovered; got %v", want, summary.Names)
		}
	}
}
