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
