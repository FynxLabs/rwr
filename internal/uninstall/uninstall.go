// Package uninstall reverses what the run journal shows was applied — and
// only that. Input is the record, never the blueprint tree: uninstall after
// editing or deleting the tree still works, and nothing without a record
// entry is ever touched.
package uninstall

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/status"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// reverseOrder is the inverse of the apply order: what was applied last is
// removed first.
var reverseOrder = []string{
	types.BlueprintTypeGit,
	types.BlueprintTypeServices,
	types.BlueprintTypeFonts,
	types.BlueprintTypeFiles,
	types.BlueprintTypePackages,
}

// NotReversible names what the journal records but uninstall will not undo,
// printed up front so the operator knows before confirming.
var NotReversible = []string{
	types.BlueprintTypeScripts,
	types.BlueprintTypeConfiguration,
	types.BlueprintTypeUsers,
	types.BlueprintTypeSSHKeys,
	types.BlueprintTypeRepositories,
}

// Item is one reversible entry with its planned removal.
type Item struct {
	Ref    state.EntryRef
	Action string // human description of the removal
}

// Plan orders the journal's unreversed applies for removal and separates
// out what cannot be reversed. No records → an explicit refusal: guessing
// removals from a tree that may have changed is how the wrong file dies.
func Plan(records []*state.RecordFile) (items []Item, skipped []string, err error) {
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("no run records under the state directory — nothing recorded, nothing to reverse")
	}
	refs := state.UnreversedApplies(records)

	byProcessor := map[string][]state.EntryRef{}
	for _, ref := range refs {
		byProcessor[ref.Entry().Processor] = append(byProcessor[ref.Entry().Processor], ref)
	}
	for _, processor := range NotReversible {
		for _, ref := range byProcessor[processor] {
			entry := ref.Entry()
			skipped = append(skipped, fmt.Sprintf("%s: %s", entry.Processor, entry.Identity["name"]))
		}
	}
	for _, processor := range reverseOrder {
		// Within a processor, reverse the recorded order too.
		group := byProcessor[processor]
		for i := len(group) - 1; i >= 0; i-- {
			items = append(items, Item{Ref: group[i], Action: describe(group[i].Entry())})
		}
	}
	return items, skipped, nil
}

func describe(entry *state.Entry) string {
	switch entry.Processor {
	case types.BlueprintTypePackages:
		return fmt.Sprintf("remove package %s via %s", entry.Identity["name"], entry.Identity["provider"])
	case types.BlueprintTypeFiles:
		return fmt.Sprintf("delete %s (hash-guarded)", entry.Identity["dest"])
	case types.BlueprintTypeGit:
		return fmt.Sprintf("delete checkout %s (skips a dirty worktree)", entry.Identity["target"])
	case types.BlueprintTypeServices:
		return fmt.Sprintf("disable service %s", entry.Identity["name"])
	case types.BlueprintTypeFonts:
		return fmt.Sprintf("remove font %s", entry.Identity["name"])
	}
	return "remove " + entry.Identity["name"]
}

// Execute runs the plan per-item: failures are logged and counted, never
// aborting the remaining work; reversed entries are marked in their records
// so a re-run retries only what failed.
func Execute(out io.Writer, items []Item, querier *status.Querier) (failed int) {
	for _, item := range items {
		entry := item.Ref.Entry()
		if system.IsDryRun() {
			helpers.Say(out, "[DRY-RUN] Would %s\n", item.Action)
			continue
		}
		outcome, err := reverse(entry, querier)
		switch {
		case err != nil:
			failed++
			log.Errorf("uninstall: %s: %v", item.Action, err)
			continue
		case outcome != "":
			helpers.Say(out, "skipped: %s — %s\n", item.Action, outcome)
			continue
		}
		helpers.Say(out, "done: %s\n", item.Action)
		entry.Reversed = true
		if err := item.Ref.File.Save(); err != nil {
			log.Warnf("marking reversal in %s: %v", item.Ref.File.Path, err)
		}
	}
	return failed
}

// reverse undoes one entry. A non-empty skip reason means the item was
// deliberately left alone; an error means the removal was tried and failed.
func reverse(entry *state.Entry, querier *status.Querier) (skipReason string, err error) {
	switch entry.Processor {
	case types.BlueprintTypePackages:
		return reversePackage(entry, querier)
	case types.BlueprintTypeFiles:
		return reverseFile(entry)
	case types.BlueprintTypeGit:
		return reverseGit(entry)
	case types.BlueprintTypeServices:
		return reverseService(entry)
	case types.BlueprintTypeFonts:
		return reverseFont(entry)
	}
	return "no reversal implemented", nil
}

// reverseFont deletes the font faces the recorded name installed into the
// recorded directory — the same glob the fonts processor's own remove action
// uses.
func reverseFont(entry *state.Entry) (string, error) {
	dir, name := entry.Identity["dir"], entry.Identity["name"]
	if dir == "" || name == "" {
		return "no recorded font directory", nil
	}
	removed := 0
	for _, ext := range []string{".ttf", ".otf"} {
		matches, err := filepath.Glob(filepath.Join(dir, name+"*"+ext))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if err := os.Remove(match); err != nil {
				return "", err
			}
			removed++
		}
	}
	if removed == 0 {
		return "already absent", nil
	}
	return "", nil
}

func reversePackage(entry *state.Entry, querier *status.Querier) (string, error) {
	name := entry.Identity["name"]
	provider, ok := system.GetProvider(entry.Identity["provider"])
	if !ok {
		return "provider not available on this system", nil
	}
	if querier.PackagePresent(provider, name) == status.Absent {
		return "already absent", nil
	}
	args := append(strings.Fields(provider.Commands.Remove), name)
	cmd := types.Command{
		Exec:     provider.BinPath,
		Args:     args,
		Elevated: provider.Elevated || entry.Elevated,
	}
	return "", system.RunCommand(cmd, false)
}

func reverseFile(entry *state.Entry) (string, error) {
	dest := entry.Identity["dest"]
	if dest == "" {
		return "no recorded destination", nil
	}
	if entry.Identity["sha256"] == "" {
		// Directories (and unhashed applies) carry no content hash; without
		// one, only an empty directory is safe to remove.
		if _, err := os.Stat(dest); err != nil {
			return "already absent", nil
		}
		if err := os.Remove(dest); err != nil {
			return "not empty or not removable — left in place", nil //nolint:nilerr // deliberate skip, not a failure
		}
		return "", nil
	}
	switch status.FileState(dest, entry.Identity["sha256"]) {
	case status.Absent:
		return "already absent", nil
	case status.Modified:
		return "modified since the recorded apply — not deleting", nil
	case status.Unknown:
		return "content unreadable — not deleting", nil
	}
	return "", os.Remove(dest)
}

func reverseGit(entry *state.Entry) (string, error) {
	target := entry.Identity["target"]
	if target == "" {
		return "no recorded checkout target", nil
	}
	if _, err := os.Stat(target); err != nil {
		return "already absent", nil
	}
	dirty, err := exec.Command("git", "-C", target, "status", "--porcelain").Output() // #nosec G204 -- target comes from rwr's own journal
	if err != nil {
		return "not a readable git worktree — not deleting", nil //nolint:nilerr // deliberate skip
	}
	if strings.TrimSpace(string(dirty)) != "" {
		return "worktree has local changes — not deleting", nil
	}
	return "", os.RemoveAll(target)
}

func reverseService(entry *state.Entry) (string, error) {
	name := entry.Identity["name"]
	if status.ServiceState(name) != status.Present {
		return "not enabled or not queryable", nil
	}
	cmd := types.Command{
		Exec:     "systemctl",
		Args:     []string{"disable", "--now", name},
		Elevated: true,
	}
	return "", system.RunCommand(cmd, false)
}
