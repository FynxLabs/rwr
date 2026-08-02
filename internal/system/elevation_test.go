package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
)

// An elevated copy has to create the target's directory with the same
// privilege as the copy: os.MkdirAll ran unelevated first, so a copy into a
// root-owned tree (files: elevated: true under /etc) died with EACCES before
// the elevated mv ever ran.
func TestCopyFile_ElevatedCreatesParentElevated(t *testing.T) {
	rec := exectest.New()
	defer SetExecutor(rec)()

	dir := t.TempDir()
	source := filepath.Join(dir, "unit.service")
	if err := os.WriteFile(source, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "etc", "systemd", "system", "unit.service")

	if err := CopyFile(source, target, true, nil); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	mkdirs := rec.Find("mkdir")
	if len(mkdirs) != 1 || !mkdirs[0].Elevated {
		t.Fatalf("mkdir calls = %v, want one elevated mkdir", mkdirs)
	}
	wantArgs := []string{"-p", "--", filepath.Dir(target)}
	for i, want := range wantArgs {
		if mkdirs[0].Args[i] != want {
			t.Errorf("mkdir args = %v, want %v", mkdirs[0].Args, wantArgs)
			break
		}
	}
	// The parent must NOT have been created unelevated on the real filesystem.
	if _, err := os.Stat(filepath.Dir(target)); !os.IsNotExist(err) {
		t.Error("target directory was created without elevation")
	}

	mvs := rec.Find("mv")
	if len(mvs) != 1 || !mvs[0].Elevated {
		t.Fatalf("mv calls = %v, want one elevated mv", mvs)
	}
}

// CopyDirectory accepted `elevated` and ignored it: every mkdir, remove and
// file write ran with the invoking user's privileges, so a directories
// blueprint with elevated: true failed on the first root-owned target.
func TestCopyDirectory_ThreadsElevationThroughEveryWrite(t *testing.T) {
	rec := exectest.New()
	defer SetExecutor(rec)()

	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := os.MkdirAll(filepath.Join(source, "conf.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "conf.d", "app.conf"), []byte("k=v\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "etc", "app")

	if err := CopyDirectory(source, target, true, false); err != nil {
		t.Fatalf("CopyDirectory: %v", err)
	}

	mkdirs := rec.Find("mkdir")
	if len(mkdirs) < 2 {
		t.Fatalf("mkdir calls = %v, want elevated mkdirs for the target tree", mkdirs)
	}
	for _, call := range mkdirs {
		if !call.Elevated {
			t.Errorf("unelevated mkdir reached the system: %v", call)
		}
	}

	mvs := rec.Find("mv")
	if len(mvs) != 1 || !mvs[0].Elevated {
		t.Fatalf("mv calls = %v, want one elevated mv for the file copy", mvs)
	}

	// Nothing may have been written to the target without elevation.
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("target tree was created without elevation")
	}
}
