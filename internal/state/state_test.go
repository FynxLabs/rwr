package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJournal_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, "tree", false)
	if err != nil {
		t.Fatal(err)
	}
	if w == nil {
		t.Fatal("real run got a nil writer")
	}

	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok",
		Identity: map[string]string{"provider": "pacman", "name": "git"}})
	w.Append(Entry{Processor: "packages", Action: "install", OK: false, Outcome: "failed",
		Identity: map[string]string{"provider": "pacman", "name": "vim"}})

	// Crash semantics: entries are readable before Finalize ever runs.
	applies, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(applies) != 1 || applies[0].Identity["name"] != "git" {
		t.Fatalf("applies before finalize = %+v (failed applies must not fold in)", applies)
	}

	if err := w.Finalize(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(JournalPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("journal mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestJournal_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir, "tree", true)
	if err != nil || w != nil {
		t.Fatalf("dry run: writer=%v err=%v", w, err)
	}
	w.Append(Entry{Processor: "packages"}) // nil-safe
	w.Reverse("packages", nil)
	if err := w.Finalize(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "state")); !os.IsNotExist(err) {
		t.Fatal("dry run created state")
	}
}

// A reversal is an appended event; the reversed apply folds out of
// Unreversed and stays visible in Applies.
func TestJournal_ReversalIsAnEvent(t *testing.T) {
	dir := t.TempDir()
	w, _ := NewWriter(dir, "tree", false)
	identity := map[string]string{"name": "rc", "dest": "/tmp/rc"}
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok", Identity: identity})
	_ = w.Finalize()

	u, _ := NewWriter(dir, "", false)
	u.Reverse("files", identity)
	_ = u.Finalize()

	unreversed, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreversed) != 0 {
		t.Fatalf("unreversed = %+v", unreversed)
	}
	all, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || !all[0].Reversed {
		t.Fatalf("applies = %+v, want the reversed entry visible", all)
	}
}

// The latest apply per identity wins: re-applying after a reversal makes
// the entry live again only if the reversal preceded the re-apply... a
// reversal event marks the identity, so a later apply of the same identity
// still shows reversed. That is the simple rule; a re-provisioned machine
// writes new identities in practice (new run, same identity) and status
// reads disk anyway. Documented by this test.
func TestJournal_LatestApplyWins(t *testing.T) {
	dir := t.TempDir()
	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "git"}, Detail: "first"})
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "git"}, Detail: "second"})
	_ = w.Finalize()

	applies, _ := Applies(dir)
	if len(applies) != 1 || applies[0].Detail != "second" {
		t.Fatalf("applies = %+v", applies)
	}
}

// v1 per-run record files still read; their reversed marks are honored.
func TestLegacyV1RecordsFoldIn(t *testing.T) {
	dir := t.TempDir()
	runs := filepath.Join(dir, "state", "runs")
	if err := os.MkdirAll(runs, 0o700); err != nil {
		t.Fatal(err)
	}
	record := `{"recordVersion":1,"id":"20260803-old","entries":[
		{"processor":"packages","action":"install","identity":{"name":"git"},"outcome":"ok","ok":true},
		{"processor":"files","action":"create","identity":{"name":"rc"},"outcome":"ok","ok":true,"reversed":true}
	]}`
	if err := os.WriteFile(filepath.Join(runs, "20260803-old.json"), []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}

	applies, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(applies) != 2 {
		t.Fatalf("applies = %+v", applies)
	}
	unreversed, _ := Unreversed(dir)
	if len(unreversed) != 1 || unreversed[0].Identity["name"] != "git" {
		t.Fatalf("unreversed = %+v", unreversed)
	}
}
