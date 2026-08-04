// Package state is rwr's run journal: an append-only record of what each run
// actually applied. Blueprints stay the source of desired state - the record
// is evidence of past applies, never an input to `rwr all`. `rwr status`
// reads it for drift and `rwr uninstall` for reversal.
package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RecordVersion is bumped when the on-disk shape changes incompatibly.
const RecordVersion = 1

// Record is one run's journal. It is rewritten after every entry, so a crash
// leaves a truthful partial record with Finalized still false.
type Record struct {
	RecordVersion int       `json:"recordVersion"`
	ID            string    `json:"id"`
	Started       time.Time `json:"started"`
	Finished      time.Time `json:"finished,omitzero"`
	Finalized     bool      `json:"finalized"`
	Location      string    `json:"location,omitempty"`
	Entries       []Entry   `json:"entries"`
}

// Entry is one applied unit of work. Identity carries enough to find the
// thing again: provider+name for packages, dest+sha256 for files, target for
// git checkouts. Reversed is set by uninstall runs - entries are never
// deleted, the journal is history.
type Entry struct {
	Processor string            `json:"processor"`
	Action    string            `json:"action,omitempty"`
	Identity  map[string]string `json:"identity"`
	Detail    string            `json:"detail,omitempty"`
	Outcome   string            `json:"outcome"`
	OK        bool              `json:"ok"`
	Elevated  bool              `json:"elevated,omitempty"`
	Reversed  bool              `json:"reversed,omitempty"`
}

// Writer appends entries to one run's record, rewriting the file after each
// append so the on-disk record is always valid JSON. The zero value is a
// no-op writer: a nil *Writer accepts appends and writes nothing, which is
// how dry-run stays journal-free without callers branching.
type Writer struct {
	mu     sync.Mutex
	path   string
	dir    string
	record Record
}

// RunsDir is where run records live under a config directory.
func RunsDir(configDir string) string {
	return filepath.Join(configDir, "state", "runs")
}

// NewWriter opens a journal for one run. dryRun returns a nil writer: the
// run must leave no record of applies that never happened.
func NewWriter(configDir, location string, dryRun bool) (*Writer, error) {
	if dryRun || configDir == "" {
		return nil, nil
	}
	dir := RunsDir(configDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}

	started := time.Now()
	shortID := make([]byte, 4)
	if _, err := rand.Read(shortID); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("%s-%s", started.Format("20060102-150405"), hex.EncodeToString(shortID))

	w := &Writer{
		path: filepath.Join(dir, id+".json"),
		dir:  dir,
		record: Record{
			RecordVersion: RecordVersion,
			ID:            id,
			Started:       started,
			Location:      location,
		},
	}
	if err := w.flush(); err != nil {
		return nil, err
	}
	return w, nil
}

// Append records one applied unit and rewrites the record file. On a nil
// writer it is a no-op.
func (w *Writer) Append(entry Entry) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.record.Entries = append(w.record.Entries, entry)
	// A failed flush must not abort the run the journal is only observing;
	// the error surfaces at Finalize, which callers do check.
	_ = w.flush() //nolint:errcheck
}

// Finalize marks the record complete and points `latest` at it.
func (w *Writer) Finalize() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.record.Finished = time.Now()
	w.record.Finalized = true
	if err := w.flush(); err != nil {
		return err
	}
	// latest is a plain file naming the newest record - a symlink breaks on
	// Windows without privileges.
	latest := filepath.Join(w.dir, "latest")
	return os.WriteFile(latest, []byte(filepath.Base(w.path)+"\n"), 0o600)
}

// flush rewrites the record via a temp file + rename, so a crash mid-write
// cannot leave truncated JSON. Callers hold w.mu.
func (w *Writer) flush() error {
	data, err := json.MarshalIndent(&w.record, "", "  ")
	if err != nil {
		return err
	}
	tmp := w.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, w.path)
}

// Load reads one record file.
func Load(path string) (*Record, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- rwr's own state directory
	if err != nil {
		return nil, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if record.RecordVersion > RecordVersion {
		return nil, fmt.Errorf("%s is record version %d; this rwr reads up to %d", path, record.RecordVersion, RecordVersion)
	}
	return &record, nil
}

// Latest resolves the newest finalized record under configDir, or nil when
// no record exists.
func Latest(configDir string) (*Record, error) {
	dir := RunsDir(configDir)
	pointer, err := os.ReadFile(filepath.Join(dir, "latest")) // #nosec G304 -- rwr's own state directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	name := filepath.Base(trimNewline(pointer))
	return Load(filepath.Join(dir, name))
}

func trimNewline(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
