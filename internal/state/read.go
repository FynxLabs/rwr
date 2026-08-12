package state

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Applies folds the journal into applied entries: OK applies only, the
// latest per identity (a re-apply wins), with Reversed computed from
// reverse events. Legacy v1 per-run record files are folded in first, so
// history recorded before the event log survives it.
func Applies(configDir string) ([]Entry, error) {
	var entries []Entry
	// applyPos[i] is the journal position of entries[i]. The log is append
	// only, so position is time: it is what lets a reversal be compared
	// against the apply it is meant to undo.
	var applyPos []int
	// reversedAt holds the position of the most recent reverse per identity,
	// not merely the fact that one exists. A reversal only cancels an apply
	// that came before it: apply, uninstall, re-apply leaves the file on disk
	// and the record has to say so.
	reversedAt := map[string]int{}
	position := 0

	legacy, err := legacyEntries(configDir)
	if err != nil {
		return nil, err
	}
	// v1 records predate the event log, so every one of them sorts before
	// every journal event.
	for _, entry := range legacy {
		entries = append(entries, entry)
		applyPos = append(applyPos, position)
		position++
	}

	file, err := os.Open(JournalPath(configDir)) // #nosec G304 -- rwr's own state directory
	if err == nil {
		defer file.Close() //nolint:errcheck
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			var event Event
			// A partial last line after a crash does not parse; every
			// complete line before it still counts.
			if json.Unmarshal(scanner.Bytes(), &event) != nil {
				continue
			}
			// Every parsed event advances position, including the run and
			// finish events, so positions stay strictly ordered.
			position++
			switch event.Kind {
			case "apply":
				if event.OK {
					entries = append(entries, Entry{
						Run: event.Run, Processor: event.Processor, Action: event.Action,
						Identity: event.Identity, Detail: event.Detail,
						Outcome: event.Outcome, OK: event.OK, Elevated: event.Elevated,
					})
					applyPos = append(applyPos, position)
				}
			case "reverse":
				reversedAt[Key(event.Processor, event.Identity)] = position
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Latest apply per identity wins. The latest entry is the one kept, so
	// its guard values (a file's sha256) are the ones consumers see - the
	// hash of the content actually on disk after the most recent apply,
	// which is what a hash-guarded delete needs.
	latest := map[string]int{}
	var order []string
	for i, entry := range entries {
		key := Key(entry.Processor, entry.Identity)
		if _, seen := latest[key]; !seen {
			order = append(order, key)
		}
		latest[key] = i
	}
	folded := make([]Entry, 0, len(order))
	for _, key := range order {
		index := latest[key]
		entry := entries[index]
		// Reversed only when the reversal came after the apply being kept.
		// Treating any reversal as cancelling the identity forever meant
		// `rwr all`, `rwr uninstall`, `rwr all` left every re-applied unit
		// looking reversed: uninstall would not offer to remove it again and
		// status did not count it, while the thing was sitting on disk.
		if pos, ok := reversedAt[key]; ok && pos > applyPos[index] {
			entry.Reversed = true
		}
		folded = append(folded, entry)
	}
	return folded, nil
}

// Unreversed is Applies minus what uninstall runs have reversed.
func Unreversed(configDir string) ([]Entry, error) {
	all, err := Applies(configDir)
	if err != nil {
		return nil, err
	}
	kept := make([]Entry, 0, len(all))
	for _, entry := range all {
		if !entry.Reversed {
			kept = append(kept, entry)
		}
	}
	return kept, nil
}

// guardKeys are identity entries that describe the state a unit was left in
// rather than which unit it is.
//
// They travel inside Identity because consumers need them next to the thing
// they guard: uninstall refuses to delete a file whose content no longer
// matches the recorded sha256. But two applies that differ only in a guard are
// the same unit, and folding has to say so.
//
// Including sha256 in the fold key meant re-applying a file with changed
// content produced a brand new identity. The previous entry never folded away
// and stayed unreversed forever, so the journal accreted one permanent entry
// per content version, `rwr uninstall` planned N deletes for one path (N-1
// reporting "already absent"), and `rwr status` could report a live file as
// stale.
var guardKeys = map[string]bool{"sha256": true}

// Key renders the identifying part of an identity deterministically, so the
// same unit keys the same way across runs and across a reversal.
//
// Fields are length-prefixed rather than delimited, because identity values
// are arbitrary strings and one of them is a filesystem path. Joining them
// with "=" and ";" was ambiguous: both of these render identically, and would
// fold two different managed files into one entry, so one of them would never
// be reversed and the other could be matched against the wrong content hash.
//
//	{"dest": "/tmp/a;name=x", "name": "y"}
//	{"dest": "/tmp/a",        "name": "x;name=y"}
//
// Length prefixes make the encoding injective: no value can imitate the
// structure around it, whatever characters it contains.
//
// The format is never written to disk. Keys are computed while folding and
// discarded, so changing it does not affect journals already recorded.
func Key(processor string, identity map[string]string) string {
	keys := make([]string, 0, len(identity))
	for k := range identity {
		if guardKeys[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	writeKeyField(&b, processor)
	for _, k := range keys {
		writeKeyField(&b, k)
		writeKeyField(&b, identity[k])
	}
	return b.String()
}

// writeKeyField appends one length-prefixed field: "<len>:<value>".
func writeKeyField(b *strings.Builder, value string) {
	b.WriteString(strconv.Itoa(len(value)))
	b.WriteByte(':')
	b.WriteString(value)
}

// legacyEntries reads the v1 per-run record files this format replaced.
// Reversed marks recorded there are honored; unreadable files are skipped.
func legacyEntries(configDir string) ([]Entry, error) {
	dir := filepath.Join(configDir, "state", "runs")
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type v1Entry struct {
		Processor string            `json:"processor"`
		Action    string            `json:"action"`
		Identity  map[string]string `json:"identity"`
		Detail    string            `json:"detail"`
		Outcome   string            `json:"outcome"`
		OK        bool              `json:"ok"`
		Elevated  bool              `json:"elevated"`
		Reversed  bool              `json:"reversed"`
	}
	type v1Record struct {
		ID      string    `json:"id"`
		Entries []v1Entry `json:"entries"`
	}

	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names) // filenames start with the timestamp; oldest first

	var entries []Entry
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- rwr's own state directory
		if err != nil {
			continue
		}
		var record v1Record
		if json.Unmarshal(data, &record) != nil {
			continue
		}
		for _, e := range record.Entries {
			if !e.OK {
				continue
			}
			entries = append(entries, Entry{
				Run: record.ID, Processor: e.Processor, Action: e.Action,
				Identity: e.Identity, Detail: e.Detail, Outcome: e.Outcome,
				OK: e.OK, Elevated: e.Elevated, Reversed: e.Reversed,
			})
		}
	}
	return entries, nil
}
