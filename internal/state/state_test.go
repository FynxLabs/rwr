package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriter_RecordLifecycle(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, "tree", false)
	if err != nil {
		t.Fatal(err)
	}
	if w == nil {
		t.Fatal("real run got a nil writer")
	}

	w.Append(Entry{Processor: "packages", Action: "install",
		Identity: map[string]string{"provider": "pacman", "name": "git"}, Outcome: "ok", OK: true})

	// Crash semantics: the on-disk record is already valid and unfinalized
	// before Finalize runs.
	partial, err := Load(w.path)
	if err != nil {
		t.Fatalf("partial record unreadable: %v", err)
	}
	if partial.Finalized {
		t.Fatal("record finalized before Finalize")
	}
	if len(partial.Entries) != 1 || partial.Entries[0].Identity["name"] != "git" {
		t.Fatalf("partial entries = %+v", partial.Entries)
	}

	if err := w.Finalize(); err != nil {
		t.Fatal(err)
	}
	latest, err := Latest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || !latest.Finalized || latest.ID != partial.ID {
		t.Fatalf("latest = %+v", latest)
	}

	// User-only permissions on the record file.
	info, err := os.Stat(w.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("record mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestWriter_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, "tree", true)
	if err != nil {
		t.Fatal(err)
	}
	if w != nil {
		t.Fatal("dry-run got a real writer")
	}
	w.Append(Entry{Processor: "packages"}) // nil-safe no-op
	if err := w.Finalize(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(RunsDir(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run created the state directory")
	}
}

func TestLatest_NoRecords(t *testing.T) {
	record, err := Latest(t.TempDir())
	if err != nil || record != nil {
		t.Fatalf("empty state dir: record=%v err=%v", record, err)
	}
}

func TestLoad_RejectsNewerVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.json")
	if err := os.WriteFile(path, []byte(`{"recordVersion": 99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("newer record version accepted")
	}
}
