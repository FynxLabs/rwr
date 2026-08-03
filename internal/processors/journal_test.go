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

	record, err := state.Latest(configDir)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || !record.Finalized {
		t.Fatalf("record = %+v", record)
	}
	if len(record.Entries) != 3 {
		t.Fatalf("entries = %d, want 3 (planned items must not be recorded): %+v", len(record.Entries), record.Entries)
	}
	if record.Entries[0].Identity["provider"] != "pacman" || record.Entries[0].Identity["name"] != "git" || !record.Entries[0].OK {
		t.Fatalf("entry 0 = %+v", record.Entries[0])
	}
	if record.Entries[1].OK || record.Entries[1].Outcome != string(types.StatusFailed) {
		t.Fatalf("entry 1 = %+v", record.Entries[1])
	}
	if record.Entries[2].Identity["dest"] != "/tmp/rc" || record.Entries[2].Identity["sha256"] != "ab" {
		t.Fatalf("entry 2 identity = %+v", record.Entries[2].Identity)
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
