// Package state is rwr's run journal: one append-only event log. Every line
// of <configdir>/state/journal.jsonl is a self-contained JSON event (a run
// starting, a unit applied, a run finishing, a reversal). Appends are O(1),
// a crash loses at most a partial last line, and nothing is ever rewritten
// or mutated. Blueprints stay the source of desired state; the journal is
// evidence of what happened.
package state

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Version marks the journal line format.
const Version = 2

// Event is one journal line. Kind decides which fields matter:
//   - "run":     ID, Started, Location
//   - "apply":   Run, Entry fields
//   - "finish":  Run, Finished
//   - "reverse": Run (the reversing run), Processor, Identity (the target)
type Event struct {
	V    int    `json:"v"`
	Kind string `json:"kind"`

	ID       string    `json:"id,omitempty"`
	Run      string    `json:"run,omitempty"`
	Started  time.Time `json:"started,omitzero"`
	Finished time.Time `json:"finished,omitzero"`
	Location string    `json:"location,omitempty"`

	Processor string            `json:"processor,omitempty"`
	Action    string            `json:"action,omitempty"`
	Identity  map[string]string `json:"identity,omitempty"`
	Detail    string            `json:"detail,omitempty"`
	Outcome   string            `json:"outcome,omitempty"`
	OK        bool              `json:"ok,omitempty"`
	Elevated  bool              `json:"elevated,omitempty"`
}

// Entry is one applied unit as consumers see it after folding the log.
type Entry struct {
	Run       string
	Processor string
	Action    string
	Identity  map[string]string
	Detail    string
	Outcome   string
	OK        bool
	Elevated  bool
	Reversed  bool
}

// JournalPath is the event log under a config directory.
func JournalPath(configDir string) string {
	return filepath.Join(configDir, "state", "journal.jsonl")
}

// Writer appends events for one run. A nil Writer accepts every call and
// writes nothing, which is how dry-run stays journal-free without callers
// branching.
type Writer struct {
	mu   sync.Mutex
	file *os.File
	run  string
}

// NewWriter opens the journal and appends this run's start event. dryRun
// (or no config dir) returns a nil writer.
func NewWriter(configDir, location string, dryRun bool) (*Writer, error) {
	if dryRun || configDir == "" {
		return nil, nil
	}
	dir := filepath.Dir(JournalPath(configDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}
	file, err := os.OpenFile(JournalPath(configDir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}

	short := make([]byte, 4)
	if _, err := rand.Read(short); err != nil {
		file.Close() //nolint:errcheck
		return nil, err
	}
	w := &Writer{file: file, run: time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(short)}
	w.append(Event{V: Version, Kind: "run", ID: w.run, Started: time.Now(), Location: location})
	return w, nil
}

// Run is this writer's run id.
func (w *Writer) Run() string {
	if w == nil {
		return ""
	}
	return w.run
}

// Append records one applied unit.
func (w *Writer) Append(entry Entry) {
	if w == nil {
		return
	}
	w.append(Event{
		V: Version, Kind: "apply", Run: w.run,
		Processor: entry.Processor, Action: entry.Action, Identity: entry.Identity,
		Detail: entry.Detail, Outcome: entry.Outcome, OK: entry.OK, Elevated: entry.Elevated,
	})
}

// Reverse records that this run reversed a previously applied unit.
func (w *Writer) Reverse(processor string, identity map[string]string) {
	if w == nil {
		return
	}
	w.append(Event{V: Version, Kind: "reverse", Run: w.run, Processor: processor, Identity: identity})
}

// Finalize appends the finish event and closes the journal.
func (w *Writer) Finalize() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writeLocked(Event{V: Version, Kind: "finish", Run: w.run, Finished: time.Now()})
	return w.file.Close()
}

func (w *Writer) append(event Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writeLocked(event)
}

// writeLocked appends one line. The journal observes the run; a failed
// write must not abort the work being recorded.
func (w *Writer) writeLocked(event Event) {
	line, err := json.Marshal(event)
	if err != nil {
		return
	}
	writer := bufio.NewWriter(w.file)
	writer.Write(line)     //nolint:errcheck
	writer.WriteByte('\n') //nolint:errcheck
	_ = writer.Flush()     //nolint:errcheck
	_ = w.file.Sync()      //nolint:errcheck
}
