package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Source says where a record came from.
type Source int

const (
	SrcLog Source = iota
	SrcStdout
	SrcStderr
)

// LogRecord is one captured line, attributed to the processor that was
// running when it was written.
type LogRecord struct {
	Seq       uint64
	Time      time.Time
	Level     string // "DEBU" "INFO" "WARN" "ERRO" "FATA"; "" for raw command output
	Processor string
	Provider  string
	Msg       string
	Caller    string // "packages.go:199"; display renders it at debug level only
	Src       Source
}

// currentProcessor is the package-level stamp the capture path reads. The
// run loop sets it around each dispatch, so every existing log call in the
// ten processor files is attributed with zero edits to those files - the
// same pattern as the executor's package state.
var currentProcessor atomic.Value

// SetCurrentProcessor stamps subsequent captured records.
func SetCurrentProcessor(name string) { currentProcessor.Store(name) }

// CurrentProcessor returns the active stamp ("" before any dispatch).
func CurrentProcessor() string {
	if v, ok := currentProcessor.Load().(string); ok {
		return v
	}
	return ""
}

// currentProvider is the provider-lane stamp, set by the loops that know
// which provider is doing the work (packages, repositories). It fills the
// provider column of the log view.
var currentProvider atomic.Value

// SetCurrentProvider stamps subsequent captured records ("" clears).
func SetCurrentProvider(name string) { currentProvider.Store(name) }

// CurrentProvider returns the active provider stamp.
func CurrentProvider() string {
	if v, ok := currentProvider.Load().(string); ok {
		return v
	}
	return ""
}

// Store is the append-only ring of captured records. Views hold Seq values,
// never indices: once the ring wraps, positions would scramble or panic,
// Seqs stay stable and stale ones are dropped lazily on read.
type Store struct {
	mu     sync.Mutex
	buf    []LogRecord
	cap    int
	seq    uint64
	oldest uint64
	file   *os.File
	views  []*View
}

// NewStore returns a ring capped at capacity records (50k is the default
// the --tui-buffer flag documents).
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 50000
	}
	return &Store{cap: capacity}
}

// AttachRunLog writes every record to path unconditionally, regardless of the
// display level; mode 0600 because command output can carry anything.
func (s *Store) AttachRunLog(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- run log path chosen by rwr or --log-file
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.file = f
	s.mu.Unlock()
	return nil
}

// Append records one line and returns its Seq. A message with embedded
// newlines (an error carrying a command's stderr) is split into one record
// per line: the display budgets height by record, so a single 20-line record
// would push the frame past the terminal and scroll the dashboard away.
func (s *Store) Append(record LogRecord) uint64 {
	if i := strings.IndexByte(record.Msg, '\n'); i >= 0 {
		lines := strings.Split(record.Msg, "\n")
		var last uint64
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			sub := record
			sub.Msg = line
			last = s.appendOne(sub)
		}
		return last
	}
	return s.appendOne(record)
}

func (s *Store) appendOne(record LogRecord) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	record.Seq = s.seq
	if record.Time.IsZero() {
		record.Time = time.Now()
	}

	if len(s.buf) == s.cap {
		s.buf = s.buf[1:]
		s.oldest++
	}
	s.buf = append(s.buf, record)

	if s.file != nil {
		// Full fidelity including level and caller - rendering rules are
		// display-only. Best-effort by design: a full disk must not take the
		// run down with the log of the run.
		level := record.Level
		if level == "" {
			level = "OUT " // raw command output
		}
		fmt.Fprintf(s.file, "%s %s [%s] %s %s\n", record.Time.Format(time.RFC3339Nano), level, record.Processor, record.Msg, record.Caller) //nolint:errcheck // a full disk must not take the run down with its own log
	}

	for _, view := range s.views {
		if view.Processor == "" || view.Processor == record.Processor {
			view.idx = append(view.idx, record.Seq)
		}
	}
	return s.seq
}

// Oldest returns the lowest Seq still in the ring (0 when empty).
func (s *Store) Oldest() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.buf) == 0 {
		return 0
	}
	return s.buf[0].Seq
}

// Get returns the record for a Seq, or false when it has been evicted.
func (s *Store) Get(seq uint64) (LogRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.buf) == 0 || seq < s.buf[0].Seq || seq > s.seq {
		return LogRecord{}, false
	}
	return s.buf[seq-s.buf[0].Seq], true
}

// View is a per-processor window into the store ("" = all records). It keeps
// its own scroll position and follow state; records are stored once and
// filtering is O(1) at render.
type View struct {
	Processor string
	idx       []uint64
	Offset    int
	Follow    bool
}

// NewView registers a filtered view.
func (s *Store) NewView(processor string) *View {
	s.mu.Lock()
	defer s.mu.Unlock()
	view := &View{Processor: processor, Follow: true}
	for _, record := range s.buf {
		if processor == "" || processor == record.Processor {
			view.idx = append(view.idx, record.Seq)
		}
	}
	s.views = append(s.views, view)
	return view
}

// Records returns the view's live records, dropping evicted Seqs lazily.
// The lock covers the whole read: appendOne mutates view.idx under s.mu, and
// the render tick calling this concurrently with the run goroutine's log
// writes would otherwise race on the slice header - stale lengths (lines
// silently missing from the viewport) or a re-slice clobbering an append.
func (s *Store) Records(view *View) []LogRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldest := uint64(0)
	if len(s.buf) > 0 {
		oldest = s.buf[0].Seq
	}

	start := 0
	for start < len(view.idx) && view.idx[start] < oldest {
		start++
	}
	view.idx = view.idx[start:]

	out := make([]LogRecord, 0, len(view.idx))
	for _, seq := range view.idx {
		if len(s.buf) == 0 || seq < s.buf[0].Seq || seq > s.seq {
			continue
		}
		out = append(out, s.buf[seq-s.buf[0].Seq])
	}
	return out
}

// LineWriter buffers to newlines and appends one record per line, stamped
// with the current processor. It is the tee installed on the log output and
// on captured command stdout/stderr.
type LineWriter struct {
	Store *Store
	Src   Source

	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer.
func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Partial line: keep it buffered for the next write.
			w.buf.WriteString(line)
			break
		}
		w.Store.Append(LogRecord{
			Processor: CurrentProcessor(),
			Provider:  CurrentProvider(),
			Msg:       line[:len(line)-1],
			Src:       w.Src,
			Level:     levelOf(line),
		})
	}
	return len(p), nil
}

// levelOf recovers the level from charm log's rendered prefix; command
// output has none and stays "".
func levelOf(line string) string {
	for _, level := range []string{"DEBU", "INFO", "WARN", "ERRO", "FATA"} {
		if bytes.Contains([]byte(line), []byte(level+" ")) {
			return level
		}
	}
	return ""
}

// JSONCaptureWriter parses charm log's JSON formatter output - one object per
// line - into structured LogRecords. This is the TUI capture path: level,
// message, timestamp, and caller survive as fields, so the viewport renders
// them itself and the text formatter's `<file:line>` and level prefixes never
// reach the screen.
type JSONCaptureWriter struct {
	Store *Store

	mu  sync.Mutex
	buf bytes.Buffer
}

// shortLevel maps the JSON formatter's level values onto the four-letter
// forms the display filters use.
var shortLevel = map[string]string{
	"debug": "DEBU", "info": "INFO", "warn": "WARN", "error": "ERRO", "fatal": "FATA",
}

// Write implements io.Writer.
func (w *JSONCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			w.buf.WriteString(line)
			break
		}
		w.Store.Append(w.parse(line[:len(line)-1]))
	}
	return len(p), nil
}

// parse decodes one JSON log line; a line that is not JSON (a stray print)
// is kept verbatim rather than dropped.
func (w *JSONCaptureWriter) parse(line string) LogRecord {
	record := LogRecord{Processor: CurrentProcessor(), Provider: CurrentProvider(), Src: SrcLog}

	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		record.Msg = line
		return record
	}

	if v, ok := obj["msg"].(string); ok {
		record.Msg = v
		delete(obj, "msg")
	}
	if v, ok := obj["level"].(string); ok {
		record.Level = shortLevel[v]
		delete(obj, "level")
	}
	if v, ok := obj["caller"].(string); ok {
		// The formatter emits a full path; the display wants file.go:line.
		record.Caller = filepath.Base(v)
		delete(obj, "caller")
	}
	if v, ok := obj["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			record.Time = t
		}
		delete(obj, "time")
	}
	// The logger's own prefix ("rwr: ") is branding, not data.
	delete(obj, "prefix")
	// Remaining structured fields append as k=v so nothing is lost.
	if len(obj) > 0 {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			record.Msg += fmt.Sprintf(" %s=%v", k, obj[k])
		}
	}
	return record
}

// commandSink, when set, receives captured stdout/stderr of non-interactive
// commands so they render in the log view (the `≫` lines at debug level).
var commandSink atomic.Pointer[Store]

// SetCommandSink routes captured command output into store (nil clears).
func SetCommandSink(store *Store) { commandSink.Store(store) }

// CommandOutputWriter returns a writer that records command output lines, or
// nil when no sink is installed (headless: output flows as it always has).
func CommandOutputWriter(src Source) io.Writer {
	store := commandSink.Load()
	if store == nil {
		return nil
	}
	return &LineWriter{Store: store, Src: src}
}
