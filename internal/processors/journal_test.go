package processors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/spf13/viper"
)

// The progress seam writes applied items into the run journal with their
// identity; planned and skipped items leave no entry.
func TestJournal_RecordsAppliesOnly(t *testing.T) {
	configDir := t.TempDir()
	viper.Set("rwr.configdir", configDir)
	defer viper.Set("rwr.configdir", "")

	openJournal("tree")
	track := newProgress(types.BlueprintTypePackages)
	track.expect("pacman", 3)
	track.item("pacman", "git", "install", types.StatusOK, "", 0)
	track.item("pacman", "vim", "install", types.StatusFailed, "boom", 0)
	track.item("pacman", "tmux", "install", types.StatusPlanned, "dry-run", 0)
	track.itemIdentity("", "rc", "create", types.StatusOK, "", 0, map[string]string{"dest": "/tmp/rc", "sha256": "ab"})
	closeJournal()

	applies, err := state.Applies(configDir)
	if err != nil {
		t.Fatal(err)
	}
	// Failed and planned items must not fold into applies; the two OK items
	// carry their identities.
	if len(applies) != 2 {
		t.Fatalf("applies = %+v, want 2", applies)
	}
	if applies[0].Identity["provider"] != "pacman" || applies[0].Identity["name"] != "git" {
		t.Fatalf("apply 0 = %+v", applies[0])
	}
	if applies[1].Identity["dest"] != "/tmp/rc" || applies[1].Identity["sha256"] != "ab" {
		t.Fatalf("apply 1 identity = %+v", applies[1])
	}
}

// fileJournalIdentity hashes the applied target so uninstall can hash-guard
// the delete.
func TestFileJournalIdentity(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(dest, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := types.File{Name: "out.txt", Target: dir, Action: types.FileActionCreate, Content: "content"}
	identity := fileJournalIdentity(file, dir)
	if identity["dest"] != dest {
		t.Fatalf("dest = %q, want %q", identity["dest"], dest)
	}
	if len(identity["sha256"]) != 64 {
		t.Fatalf("sha256 = %q, want a hex digest", identity["sha256"])
	}
}
