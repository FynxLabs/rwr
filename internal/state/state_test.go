package state

import (
	"os"
	"path/filepath"
	"testing"
)

// newWriter opens a journal for a test and fails loudly when it cannot.
//
// The tests used to discard the error from NewWriter and from Finalize. A
// setup failure then produced an empty journal, Applies returned nothing, and
// an assertion like "want nothing left to reverse" passed without any
// reversal having been tested.
func newWriter(t *testing.T, dir, location string) *Writer {
	t.Helper()
	w, err := NewWriter(dir, location, false)
	if err != nil {
		t.Fatalf("opening the journal: %v", err)
	}
	if w == nil {
		t.Fatal("opening the journal returned no writer")
	}
	return w
}

// finalize closes a journal and fails when the close does not land, so a
// half-written log cannot masquerade as an empty one.
func finalize(t *testing.T, w *Writer) {
	t.Helper()
	if err := w.Finalize(); err != nil {
		t.Fatalf("finalizing the journal: %v", err)
	}
}

func TestJournal_Lifecycle(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	dir := t.TempDir()
	w := newWriter(t, dir, "tree")
	identity := map[string]string{"name": "rc", "dest": "/tmp/rc"}
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok", Identity: identity})
	finalize(t, w)

	u := newWriter(t, dir, "")
	u.Reverse("files", identity)
	finalize(t, u)

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
	t.Parallel()

	dir := t.TempDir()
	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "git"}, Detail: "first"})
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "git"}, Detail: "second"})
	finalize(t, w)

	applies, err := Applies(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(applies) != 1 || applies[0].Detail != "second" {
		t.Fatalf("applies = %+v", applies)
	}
}

// v1 per-run record files still read; their reversed marks are honored.
func TestLegacyV1RecordsFoldIn(t *testing.T) {
	t.Parallel()

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
	unreversed, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
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
	t.Parallel()

	dir := t.TempDir()
	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}, Detail: "first"})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}, Detail: "second"})
	finalize(t, w)

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
	t.Parallel()

	dir := t.TempDir()
	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}})
	finalize(t, w)

	u := newWriter(t, dir, "")
	// uninstall reverses what it read: the latest identity, hash included.
	u.Reverse("files", map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"})
	finalize(t, u)

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
	t.Parallel()

	dir := t.TempDir()
	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/one", "sha256": "aaa"}})
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/two", "sha256": "aaa"}})
	finalize(t, w)

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
	t.Parallel()

	dir := t.TempDir()
	identity := map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "aaa"}

	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok", Identity: identity})
	finalize(t, w)

	u := newWriter(t, dir, "")
	u.Reverse("files", identity)
	finalize(t, u)

	again := newWriter(t, dir, "tree")
	again.Append(Entry{Processor: "files", Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": "rc", "dest": "/tmp/rc", "sha256": "bbb"}})
	finalize(t, again)

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
	t.Parallel()

	dir := t.TempDir()
	identity := map[string]string{"name": "git", "provider": "pacman"}

	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	finalize(t, w)

	u := newWriter(t, dir, "")
	u.Reverse("packages", identity)
	finalize(t, u)

	again := newWriter(t, dir, "tree")
	again.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	finalize(t, again)

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
	t.Parallel()

	dir := t.TempDir()
	identity := map[string]string{"name": "git", "provider": "pacman"}

	w := newWriter(t, dir, "tree")
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	w.Append(Entry{Processor: "packages", Action: "install", OK: true, Outcome: "ok", Identity: identity})
	finalize(t, w)

	u := newWriter(t, dir, "")
	u.Reverse("packages", identity)
	finalize(t, u)

	live, err := Unreversed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 0 {
		t.Fatalf("a reversal after the last apply did not cancel it: %+v", live)
	}
}

// Identity values are arbitrary strings, and one of them is a filesystem path.
// Joining fields with "=" and ";" was ambiguous: these two identities rendered
// identically, which would fold two different managed files into one entry -
// so one of them would never be reversed, and the other could be matched
// against the wrong content hash on the way to a delete.
func TestKey_DelimiterValuesDoNotCollide(t *testing.T) {
	t.Parallel()

	a := Key("files", map[string]string{"dest": "/tmp/a;name=x", "name": "y"})
	b := Key("files", map[string]string{"dest": "/tmp/a", "name": "x;name=y"})

	if a == b {
		t.Fatalf("two distinct files collide onto one key: %q", a)
	}
}

// The same ambiguity through the processor, and through a value that imitates
// a length prefix rather than a delimiter.
func TestKey_StructuralImitationDoesNotCollide(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b string
	}{
		{
			name: "processor absorbs a field",
			a:    Key("files", map[string]string{"name": "x"}),
			b:    Key("files\x004:name", map[string]string{}),
		},
		{
			name: "value imitates a length prefix",
			a:    Key("files", map[string]string{"name": "4:dest"}),
			b:    Key("files", map[string]string{"name": "", "4": "dest"}),
		},
	}

	for _, tt := range cases {
		if tt.a == tt.b {
			t.Errorf("%s: distinct identities collide onto %q", tt.name, tt.a)
		}
	}
}

// Keying has to be stable across calls, or the fold would scatter one unit
// across several entries.
func TestKey_IsStableAcrossCalls(t *testing.T) {
	t.Parallel()

	identity := map[string]string{"name": "rc", "dest": "/tmp/rc", "provider": "", "sha256": "aaa"}
	first := Key("files", identity)

	for range 10 {
		if got := Key("files", identity); got != first {
			t.Fatalf("Key is not deterministic: %q then %q", first, got)
		}
	}

	// The guard is excluded, so a content change keys the same.
	changed := map[string]string{"name": "rc", "dest": "/tmp/rc", "provider": "", "sha256": "bbb"}
	if Key("files", changed) != first {
		t.Error("a content change altered the identity key")
	}
}
