package state

import (
	"os"
	"path/filepath"
	"sort"
)

// RecordFile pairs a loaded record with its path so consumers can write
// entry updates (reversal marks) back where they came from.
type RecordFile struct {
	Path   string
	Record *Record
}

// Save rewrites the record file in place (temp + rename, like the writer).
func (r *RecordFile) Save() error {
	w := &Writer{path: r.Path, record: *r.Record}
	return w.flush()
}

// LoadAll reads every run record under configDir, oldest first. Unreadable
// files are skipped - one corrupt record must not hide the others.
func LoadAll(configDir string) ([]*RecordFile, error) {
	dir := RunsDir(configDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []*RecordFile
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		record, err := Load(path)
		if err != nil {
			continue
		}
		records = append(records, &RecordFile{Path: path, Record: record})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Record.Started.Before(records[j].Record.Started)
	})
	return records, nil
}

// EntryRef points at one entry inside one record file, so a reversal can be
// marked exactly where it was recorded.
type EntryRef struct {
	File  *RecordFile
	Index int
}

// Entry returns the referenced entry.
func (ref EntryRef) Entry() *Entry { return &ref.File.Record.Entries[ref.Index] }

// UnreversedApplies returns every successfully applied, not-yet-reversed
// entry across records, one per identity (the latest apply wins - that is
// the state on disk). Uninstall consumes this: reversed work must not be
// reversed again.
func UnreversedApplies(records []*RecordFile) []EntryRef {
	return latestApplies(records, false)
}

// LatestApplies is UnreversedApplies including reversed entries. Status
// enrichment consumes this: a reversal does not erase the knowledge of
// where an identity lives on disk.
func LatestApplies(records []*RecordFile) []EntryRef {
	return latestApplies(records, true)
}

func latestApplies(records []*RecordFile, includeReversed bool) []EntryRef {
	type key struct{ processor, identity string }
	latest := map[key]EntryRef{}
	var order []key
	for _, file := range records { // oldest → newest, so later applies win
		for i := range file.Record.Entries {
			entry := &file.Record.Entries[i]
			if !entry.OK || (entry.Reversed && !includeReversed) {
				continue
			}
			k := key{entry.Processor, identityString(entry.Identity)}
			if _, seen := latest[k]; !seen {
				order = append(order, k)
			}
			latest[k] = EntryRef{File: file, Index: i}
		}
	}
	refs := make([]EntryRef, 0, len(order))
	for _, k := range order {
		refs = append(refs, latest[k])
	}
	return refs
}

// identityString renders an identity map deterministically for keying.
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
