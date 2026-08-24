package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// The walk must not route the init file or bootstrap into a processor bucket:
// init configures the run and bootstrap is dispatched separately, so both used
// to land in the dead "." bucket alongside genuinely misplaced files.
func TestGetBlueprintFileOrder_SkipsInitAndBootstrap(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"init.yaml", "bootstrap.yaml", filepath.Join("packages", "base.yaml")} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("packages: []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	initConfig := &types.InitConfig{}
	initConfig.Init.Format = "yaml"

	fileOrder, err := GetBlueprintFileOrder(dir, nil, false, initConfig)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder: %v", err)
	}

	if got := fileOrder[types.BlueprintTypePackages]; len(got) != 1 || got[0] != filepath.Join("packages", "base.yaml") {
		t.Errorf("packages bucket = %v, want [packages/base.yaml]", got)
	}
	for bucket, files := range fileOrder {
		for _, f := range files {
			base := filepath.Base(f)
			if base == "init.yaml" || base == "bootstrap.yaml" {
				t.Errorf("%s routed into bucket %q; init/bootstrap must not be walked", f, bucket)
			}
		}
	}
}

// The run-order walk sees every registered format, not only files matching the
// tree-wide Init.Format.
func TestGetBlueprintFileOrder_MixedFormatTree(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"packages/base.yaml", "files/conf.toml", "services/svc.json"} {
		path := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	initConfig := &types.InitConfig{}
	initConfig.Init.Format = "yaml"

	fileOrder, err := GetBlueprintFileOrder(dir, nil, false, initConfig)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder: %v", err)
	}
	for bucket, wantFile := range map[string]string{
		types.BlueprintTypePackages: filepath.Join("packages", "base.yaml"),
		types.BlueprintTypeFiles:    filepath.Join("files", "conf.toml"),
		types.BlueprintTypeServices: filepath.Join("services", "svc.json"),
	} {
		got := fileOrder[bucket]
		if len(got) != 1 || got[0] != wantFile {
			t.Errorf("bucket %s = %v, want [%s]", bucket, got, wantFile)
		}
	}
}

func TestGetBlueprintFileOrder_SkipsProcessorSourcePayloads(t *testing.T) {
	dir := t.TempDir()
	blueprint := filepath.Join(dir, "files", "files.cue")
	payload := filepath.Join(dir, "files", "src", "Library", "Application Support", "Code", "User", "keybindings.json")
	for path, content := range map[string]string{
		blueprint: "{files: []}\n",
		payload:   "// JSON with comments is application data, not a blueprint\n[]\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	config := &types.InitConfig{}
	config.Init.Format = "cue"
	order, err := GetBlueprintFileOrder(dir, nil, false, config)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder: %v", err)
	}
	got := order[types.BlueprintTypeFiles]
	if len(got) != 1 || got[0] != filepath.Join("files", "files.cue") {
		t.Fatalf("files bucket = %v, want only files/files.cue", got)
	}
}

// A flattened tree - blueprint files at the root, no processor directories -
// is routed by content: a file with a single recognized top-level key executes
// under that processor. This is the layout examples/alternative_layouts ships,
// which used to land in a dead bucket and exit 0 having executed nothing.
func TestGetBlueprintFileOrder_ContentBasedDetection(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("packages.yaml", "packages:\n  - name: git\n    action: install\n")
	write("everything.yaml", "services:\n  - name: sshd\n    action: enable\n")
	// Multi-type: routes to every type it declares (the minimal_files layout).
	write("mixed.yaml", "packages: []\nservices: []\n")

	initConfig := &types.InitConfig{}
	initConfig.Init.Format = "yaml"
	initConfig.Variables.UserDefined = map[string]interface{}{}

	fileOrder, err := GetBlueprintFileOrder(dir, nil, false, initConfig)
	if err != nil {
		t.Fatalf("GetBlueprintFileOrder: %v", err)
	}

	has := func(bucket, file string) bool {
		for _, f := range fileOrder[bucket] {
			if f == file {
				return true
			}
		}
		return false
	}
	if !has(types.BlueprintTypePackages, "packages.yaml") {
		t.Errorf("packages bucket = %v, want packages.yaml routed", fileOrder[types.BlueprintTypePackages])
	}
	if !has(types.BlueprintTypeServices, "everything.yaml") {
		t.Errorf("services bucket = %v, want everything.yaml routed", fileOrder[types.BlueprintTypeServices])
	}
	// The multi-type file appears in BOTH buckets; the dispatch subsets it.
	if !has(types.BlueprintTypePackages, "mixed.yaml") || !has(types.BlueprintTypeServices, "mixed.yaml") {
		t.Errorf("mixed.yaml not routed to both buckets: packages=%v services=%v",
			fileOrder[types.BlueprintTypePackages], fileOrder[types.BlueprintTypeServices])
	}
}
