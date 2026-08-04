package status

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

func TestFileState(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "f")
	if err := os.WriteFile(dest, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := system.HashFileSHA256(dest)
	if err != nil {
		t.Fatal(err)
	}

	if got := FileState(dest, sum); got != Present {
		t.Fatalf("matching hash = %s, want present", got)
	}
	if got := FileState(dest, "deadbeef"); got != Modified {
		t.Fatalf("differing hash = %s, want modified", got)
	}
	if got := FileState(filepath.Join(dir, "gone"), sum); got != Absent {
		t.Fatalf("missing file = %s, want absent", got)
	}
	if got := FileState(dest, ""); got != Present {
		t.Fatalf("no recorded hash on existing file = %s, want present", got)
	}
	if got := FileState("", "x"); got != Unknown {
		t.Fatalf("empty dest = %s, want unknown", got)
	}
}

// The package query never guesses: a provider without a list command is
// unknown, and the parse strips dpkg's :arch qualifier.
func TestPackagePresent(t *testing.T) {
	q := NewQuerier()
	if got := q.PackagePresent(&types.Provider{Name: "x"}, "git"); got != Unknown {
		t.Fatalf("no list command = %s, want unknown", got)
	}

	// A fake provider whose list binary is a script emitting dpkg-style
	// output; the binary is resolved via the first list field.
	dir := t.TempDir()
	fake := filepath.Join(dir, "fakelist")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf 'git:amd64\\tinstall\\nvim\\tinstall\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	provider := &types.Provider{Name: "fake", BinPath: "/nonexistent"}
	provider.Commands.List = fake

	if got := q.PackagePresent(provider, "git"); got != Present {
		t.Fatalf("listed package = %s, want present", got)
	}
	if got := q.PackagePresent(provider, "tmux"); got != Absent {
		t.Fatalf("unlisted package = %s, want absent", got)
	}
}

// Queries must never run a mutating command: the querier only ever executes
// the provider's list verb, and a provider with none stays unqueried.
func TestQueryNeverRunsNonListCommands(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "mutated")
	trap := filepath.Join(dir, "trap")
	if err := os.WriteFile(trap, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	provider := &types.Provider{Name: "trap", BinPath: trap}
	provider.Commands.Install = "install"
	provider.Commands.Remove = "remove"
	// No List command: the querier must not fall back to anything else.
	if got := NewQuerier().PackagePresent(provider, "git"); got != Unknown {
		t.Fatalf("got %s, want unknown", got)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("query executed a non-list command")
	}
}

func TestRowsClassification(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present")
	if err := os.WriteFile(present, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, _ := system.HashFileSHA256(present)

	plan := &types.Plan{Resources: []types.Resource{
		{Processor: types.BlueprintTypeFiles, Name: "present"},
		{Processor: types.BlueprintTypeFiles, Name: "missing"},
		{Processor: types.BlueprintTypeScripts, Name: "setup.sh"},
	}}
	applies := []state.Entry{
		{Processor: types.BlueprintTypeFiles, OK: true, Outcome: "ok",
			Identity: map[string]string{"name": "present", "dest": present, "sha256": sum}},
		{Processor: types.BlueprintTypeFiles, OK: true, Outcome: "ok",
			Identity: map[string]string{"name": "missing", "dest": filepath.Join(dir, "gone"), "sha256": sum}},
		{Processor: types.BlueprintTypeFiles, OK: true, Outcome: "ok",
			Identity: map[string]string{"name": "stale-one", "dest": present, "sha256": sum}},
	}

	rows := Rows(plan, applies, NewQuerier())
	byName := map[string]Row{}
	for _, row := range rows {
		byName[row.Name] = row
	}
	if byName["present"].Class != InSync {
		t.Errorf("present = %s", byName["present"].Class)
	}
	if byName["missing"].Class != Missing {
		t.Errorf("missing = %s", byName["missing"].Class)
	}
	if byName["setup.sh"].Class != UnknownItem {
		t.Errorf("script = %s, want unknown (not queryable)", byName["setup.sh"].Class)
	}
	if byName["stale-one"].Class != Stale {
		t.Errorf("stale-one = %s, want stale", byName["stale-one"].Class)
	}
	if !Drifted(rows) {
		t.Error("missing+stale rows did not count as drift")
	}
}
