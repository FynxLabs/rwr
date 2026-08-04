package reporting

import (
	"bytes"
	"fmt"
	"os"
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
	Level     string
	Processor string
	Provider  string
	Msg       string
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

// Append records one line and returns its Seq.
func (s *Store) Append(record LogRecord) uint64 {
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
		// Best-effort by design: a full disk must not take the run down with
		// the log of the run.
		fmt.Fprintf(s.file, "%s [%s] %s\n", record.Time.Format(time.RFC3339Nano), record.Processor, record.Msg) //nolint:errcheck // a full disk must not take the run down with its own log
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
func (s *Store) Records(view *View) []LogRecord {
	s.mu.Lock()
	oldest := uint64(0)
	if len(s.buf) > 0 {
		oldest = s.buf[0].Seq
	}
	s.mu.Unlock()

	start := 0
	for start < len(view.idx) && view.idx[start] < oldest {
		start++
	}
	view.idx = view.idx[start:]

	out := make([]LogRecord, 0, len(view.idx))
	for _, seq := range view.idx {
		if record, ok := s.Get(seq); ok {
			out = append(out, record)
		}
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
