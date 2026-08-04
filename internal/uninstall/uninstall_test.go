package uninstall

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/status"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

func fileEntry(name, dest, sha string) state.Entry {
	return state.Entry{Processor: types.BlueprintTypeFiles, Action: "create", OK: true, Outcome: "ok",
		Identity: map[string]string{"name": name, "dest": dest, "sha256": sha}}
}

func TestPlan_RefusesWithoutRecords(t *testing.T) {
	if _, _, err := Plan(nil); err == nil {
		t.Fatal("no records did not refuse")
	}
}

func TestPlan_ReverseOrderAndNotReversible(t *testing.T) {
	entries := []state.Entry{
		{Processor: types.BlueprintTypePackages, OK: true, Identity: map[string]string{"name": "git", "provider": "pacman"}},
		fileEntry("rc", "/tmp/rc", "ab"),
		{Processor: types.BlueprintTypeScripts, OK: true, Identity: map[string]string{"name": "setup.sh"}},
		{Processor: types.BlueprintTypeGit, OK: true, Identity: map[string]string{"name": "dotfiles", "target": "/tmp/dots"}},
	}
	items, skipped, err := Plan(entries)
	if err != nil {
		t.Fatal(err)
	}
	// Applied order was packages→files→git; removal must be git→files→packages.
	var processors []string
	for _, item := range items {
		processors = append(processors, item.Entry.Processor)
	}
	want := []string{types.BlueprintTypeGit, types.BlueprintTypeFiles, types.BlueprintTypePackages}
	if strings.Join(processors, ",") != strings.Join(want, ",") {
		t.Fatalf("removal order = %v, want %v", processors, want)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "setup.sh") {
		t.Fatalf("not-reversible list = %v", skipped)
	}
}

func TestExecute_HashGuardAndAbsentSkip(t *testing.T) {
	dir := t.TempDir()

	intact := filepath.Join(dir, "intact")
	if err := os.WriteFile(intact, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	intactSum, _ := system.HashFileSHA256(intact)

	modified := filepath.Join(dir, "modified")
	if err := os.WriteFile(modified, []byte("changed since apply"), 0o644); err != nil {
		t.Fatal(err)
	}

	configDir := t.TempDir()
	journal, err := state.NewWriter(configDir, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	journal.Append(fileEntry("intact", intact, intactSum))
	journal.Append(fileEntry("modified", modified, "0000000000000000000000000000000000000000000000000000000000000000"))
	journal.Append(fileEntry("gone", filepath.Join(dir, "gone"), intactSum))
	if err := journal.Finalize(); err != nil {
		t.Fatal(err)
	}

	entries, err := state.Unreversed(configDir)
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := Plan(entries)
	if err != nil {
		t.Fatal(err)
	}
	reversalJournal, err := state.NewWriter(configDir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	failed := Execute(&out, items, status.NewQuerier(), reversalJournal)
	if err := reversalJournal.Finalize(); err != nil {
		t.Fatal(err)
	}
	if failed != 0 {
		t.Fatalf("failed = %d, output:\n%s", failed, out.String())
	}

	if _, err := os.Stat(intact); !os.IsNotExist(err) {
		t.Error("hash-matching file not deleted")
	}
	if _, err := os.Stat(modified); err != nil {
		t.Error("modified file was deleted despite the hash guard")
	}
	if !strings.Contains(out.String(), "modified since the recorded apply") {
		t.Errorf("modified skip not listed:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "already absent") {
		t.Errorf("absent skip not listed:\n%s", out.String())
	}

	// Reversals are journal events: the intact delete folds out of
	// Unreversed, the modified one stays so a re-run retries it.
	remaining, err := state.Unreversed(configDir)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range remaining {
		names[entry.Identity["name"]] = true
	}
	if names["intact"] {
		t.Error("deleted entry still unreversed")
	}
	if !names["modified"] {
		t.Error("skipped entry reversed - a re-run would never retry it")
	}
}

func TestExecute_DryRunTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "f")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, _ := system.HashFileSHA256(dest)
	items, _, err := Plan([]state.Entry{fileEntry("f", dest, sum)})
	if err != nil {
		t.Fatal(err)
	}

	system.SetDryRun(true)
	defer system.SetDryRun(false)
	var out bytes.Buffer
	if failed := Execute(&out, items, status.NewQuerier(), nil); failed != 0 {
		t.Fatal("dry-run failed")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal("dry-run deleted the file")
	}
	if !strings.Contains(out.String(), "[DRY-RUN]") {
		t.Fatalf("dry-run plan not shown:\n%s", out.String())
	}
}

func TestReverseGit_DirtyWorktreeSkipped(t *testing.T) {
	entry := state.Entry{Processor: types.BlueprintTypeGit,
		Identity: map[string]string{"name": "x", "target": t.TempDir()}}
	// A plain directory is not a readable git worktree: skip, never delete.
	reason, err := reverseGit(entry)
	if err != nil || reason == "" {
		t.Fatalf("non-worktree dir: reason=%q err=%v", reason, err)
	}
}
