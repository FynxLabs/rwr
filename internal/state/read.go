package state

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Applies folds the journal into applied entries: OK applies only, the
// latest per identity (a re-apply wins), with Reversed computed from
// reverse events. Legacy v1 per-run record files are folded in first, so
// history recorded before the event log survives it.
func Applies(configDir string) ([]Entry, error) {
	var entries []Entry
	reversed := map[string]bool{}

	legacy, err := legacyEntries(configDir)
	if err != nil {
		return nil, err
	}
	entries = append(entries, legacy...)

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
			switch event.Kind {
			case "apply":
				if event.OK {
					entries = append(entries, Entry{
						Run: event.Run, Processor: event.Processor, Action: event.Action,
						Identity: event.Identity, Detail: event.Detail,
						Outcome: event.Outcome, OK: event.OK, Elevated: event.Elevated,
					})
				}
			case "reverse":
				reversed[event.Processor+"\x00"+identityString(event.Identity)] = true
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Latest apply per identity wins; reversal marks apply to the identity.
	latest := map[string]int{}
	var order []string
	for i, entry := range entries {
		key := entry.Processor + "\x00" + identityString(entry.Identity)
		if _, seen := latest[key]; !seen {
			order = append(order, key)
		}
		latest[key] = i
	}
	folded := make([]Entry, 0, len(order))
	for _, key := range order {
		entry := entries[latest[key]]
		entry.Reversed = entry.Reversed || reversed[entry.Processor+"\x00"+identityString(entry.Identity)]
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

// identityString renders an identity deterministically for keying.
func identityString(identity map[string]string) string {
	keys := make([]string, 0, len(identity))
	for k := range identity {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := ""
	for _, k := range keys {
		s += k + "=" + identity[k] + ";"
	}
	return s
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
