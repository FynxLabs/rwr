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

// The latest apply per identity wins.
//
// This used to carry a caveat: a reversal marked the identity whatever the
// order, so a later apply of the same identity still showed reversed, on the
// reasoning that a re-provisioned machine writes new identities anyway. It
// does not - a re-run writes the same identity - so that rule hid every
// re-applied unit after an uninstall. Reversal is now ordered against the
// apply it cancels; see TestJournal_ReapplyAfterReversalIsLiveAgain.
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

// A file re-applied with changed content is the same file, not a second one.
//
// The identity a files apply records is {name, dest, sha256}. Folding on the
// whole map meant a content change produced a brand new key: the previous
// entry never folded away and stayed unreversed forever. The journal accreted
// one permanent entry per content version, `rwr uninstall` planned a delete
// for each of them, and `rwr status` could call a live file stale.
func TestJournal_FileReapplyWithNewContentFoldsToOneEntry(t *testing.T) {
	dir := t.TempDir()
	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}, Detail: "first"})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}, Detail: "second"})
	_ = w.Finalize()

	applies, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(applies) != 1 {
		t.Fatalf("applies = %d entries, want 1 folded entry: %+v", len(applies), applies)
	}
	if applies[0].Detail != "second" {
		t.Errorf("kept the wrong apply: %+v", applies[0])
	}
	// The surviving entry carries the latest guard, which is what a
	// hash-guarded delete has to compare against.
	if got := applies[0].Identity["sha256"]; got != "bbb" {
		t.Errorf("sha256 = %q, want the most recent apply's hash", got)
	}
}

// Reversal has to match across a content change too: uninstall records the
// identity it saw, and a re-apply before that reversal must not leave a
// stale-hashed twin behind that looks unreversed.
func TestJournal_ReversalMatchesAcrossAContentChange(t *testing.T) {
	dir := t.TempDir()
	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}})
	_ = w.Finalize()

	u, _ := NewWriter(dir, "", false)
	// uninstall reverses what it read: the latest identity, hash included.
	u.Reverse("files", map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"})
	_ = u.Finalize()

	unreversed, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreversed) != 0 {
		t.Fatalf("unreversed = %+v, want nothing left to reverse", unreversed)
	}
}

// Two genuinely different files stay two units: dropping the guard from the
// key must not collapse anything that is actually distinct.
func TestJournal_DistinctDestinationsStayDistinct(t *testing.T) {
	dir := t.TempDir()
	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/one", "sha256": "aaa"}})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/two", "sha256": "aaa"}})
	_ = w.Finalize()

	applies, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(applies) != 2 {
		t.Fatalf("applies = %d, want 2 distinct destinations: %+v", len(applies), applies)
	}
}

// apply, uninstall, apply again: the file is on disk, and the record has to
// say so.
//
// A reversal used to cancel an identity forever, whatever order the events
// arrived in. Re-provisioning after an uninstall therefore left every
// re-applied unit looking reversed: `rwr uninstall` would not offer to remove
// it a second time and `rwr status` did not count it, while the thing was
// sitting right there. Files escaped this by accident, because their identity
// carried a content hash that made each apply a different key; folding on the
// destination removed that accident, so the ordering has to be real.
func TestJournal_ReapplyAfterReversalIsLiveAgain(t *testing.T) {
	dir := t.TempDir()
	identity := map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}

	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok", Identity: identity})
	_ = w.Finalize()

	u, _ := NewWriter(dir, "", false)
	u.Reverse("files", identity)
	_ = u.Finalize()

	again, _ := NewWriter(dir, "tree", false)
	again.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}})
	_ = again.Finalize()

	live, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 1 {
		t.Fatalf("the re-applied file is hidden by the earlier reversal: %+v", live)
	}
	if got := live[0].Identity["sha256"]; got != "bbb" {
		t.Errorf("sha256 = %q, want the re-applied content", got)
	}
}

// The same ordering rule for an identity that carries no guard at all, which
// is every processor other than files. This case was wrong before the fold
// keyed on identifying fields too, so it is not a files-only concern.
func TestJournal_ReapplyAfterReversalIsLiveAgainForPackages(t *testing.T) {
	dir := t.TempDir()
	identity := map[string]string{"name": "git", "provider": "pacman"}

	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	_ = w.Finalize()

	u, _ := NewWriter(dir, "", false)
	u.Reverse("packages", identity)
	_ = u.Finalize()

	again, _ := NewWriter(dir, "tree", false)
	again.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	_ = again.Finalize()

	live, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 1 {
		t.Fatalf("the reinstalled package is hidden by the earlier reversal: %+v", live)
	}
}

// The other direction still holds: a reversal that comes after the last apply
// really does cancel it, or uninstall would keep offering to remove things it
// already removed.
func TestJournal_ReversalAfterTheLastApplyStillCancelsIt(t *testing.T) {
	dir := t.TempDir()
	identity := map[string]string{"name": "git", "provider": "pacman"}

	w, _ := NewWriter(dir, "tree", false)
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	_ = w.Finalize()

	u, _ := NewWriter(dir, "", false)
	u.Reverse("packages", identity)
	_ = u.Finalize()

	live, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 0 {
		t.Fatalf("a reversal after the last apply did not cancel it: %+v", live)
	}
}
