package system

import (
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/types"
)

// PackageManagerInfo.Clean is stored pre-joined as "<bin> <clean args>". It used
// to be assigned straight to Command.Exec and word-split by the shell; with argv
// execution it has to be split back into a program and its arguments, or the whole
// string would be treated as one executable name and fail with ENOENT.
func TestCleanPackageManagers_SplitsCleanCommandIntoArgv(t *testing.T) {
	rec := exectest.New()
	defer SetExecutor(rec)()

	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"pacman": {
			Name:     "pacman",
			Bin:      "/usr/bin/pacman",
			Clean:    "/usr/bin/pacman -Sc --noconfirm",
			Elevated: true,
		},
	}

	if err := CleanPackageManagers(osInfo, &types.InitConfig{}); err != nil {
		t.Fatalf("CleanPackageManagers: %v", err)
	}

	if len(rec.Calls) != 1 {
		t.Fatalf("recorded %d calls, want 1: %v", len(rec.Calls), rec.Calls)
	}

	got := rec.Calls[0]
	if got.Exec != "/usr/bin/pacman" {
		t.Errorf("Exec = %q, want %q", got.Exec, "/usr/bin/pacman")
	}
	want := []string{"-Sc", "--noconfirm"}
	if len(got.Args) != len(want) {
		t.Fatalf("Args = %#v, want %#v", got.Args, want)
	}
	for i := range want {
		if got.Args[i] != want[i] {
			t.Fatalf("Args = %#v, want %#v", got.Args, want)
		}
	}
	if !got.Elevated {
		t.Error("clean command should stay elevated")
	}
}

// A manager with no clean command must not produce a call at all.
func TestCleanPackageManagers_SkipsManagersWithoutCleanCommand(t *testing.T) {
	rec := exectest.New()
	defer SetExecutor(rec)()

	osInfo := &types.OSInfo{}
	osInfo.PackageManager.Managers = map[string]types.PackageManagerInfo{
		"mas": {Name: "mas", Bin: "/usr/local/bin/mas"},
	}

	if err := CleanPackageManagers(osInfo, &types.InitConfig{}); err != nil {
		t.Fatalf("CleanPackageManagers: %v", err)
	}

	if len(rec.Calls) != 0 {
		t.Errorf("recorded %d calls, want 0: %v", len(rec.Calls), rec.Calls)
	}
}
