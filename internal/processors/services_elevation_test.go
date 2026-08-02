package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// services: action: create with inline content wrote via a plain os.WriteFile
// on all three platforms, ignoring `elevated` — a unit file under /etc/systemd
// died with EACCES for any non-root run that declared elevated: true.
func TestCreateServiceFile_ContentHonorsElevated(t *testing.T) {
	rec := exectest.New()
	defer system.SetExecutor(rec)()

	dir := t.TempDir()
	target := filepath.Join(dir, "etc", "systemd", "system", "app.service")

	err := createServiceFile(types.Service{
		Name:     "app",
		Action:   "create",
		Target:   target,
		Content:  "[Unit]\nDescription=app\n",
		Elevated: true,
	}, &types.OSInfo{})
	if err != nil {
		t.Fatalf("createServiceFile: %v", err)
	}

	// The elevated write goes through an elevated mv of a staged file; nothing
	// may have landed at the target with the invoking user's privileges.
	mvs := rec.Find("mv")
	if len(mvs) != 1 || !mvs[0].Elevated {
		t.Fatalf("mv calls = %v, want one elevated mv", mvs)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Error("service file was written without elevation")
	}
}
