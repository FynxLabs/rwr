package reporting

import (
	"strings"
	"testing"
)

// The TUI capture path parses charm log's JSON formatter output into
// structured records: level, message, and caller survive as fields, and the
// text formatter's `<file:line>` / `rwr: :` artifacts can never reach the
// screen because the screen renders from these fields.
func TestJSONCaptureWriter_ParsesFields(t *testing.T) {
	store := NewStore(100)
	w := &JSONCaptureWriter{Store: store}
	SetCurrentProcessor("packages")
	SetCurrentProvider("brew")
	defer SetCurrentProcessor("")
	defer SetCurrentProvider("")

	line := `{"time":"2026/08/11 13:30:00","level":"info","caller":"processors/packages.go:199","msg":"Successfully installed package jq via brew"}` + "\n"
	if _, err := w.Write([]byte(line)); err != nil {
		t.Fatal(err)
	}

	view := store.NewView("")
	records := store.Records(view)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	r := records[0]
	if r.Level != "INFO" {
		t.Fatalf("level = %q, want INFO", r.Level)
	}
	if r.Msg != "Successfully installed package jq via brew" {
		t.Fatalf("msg = %q", r.Msg)
	}
	if r.Caller != "packages.go:199" {
		t.Fatalf("caller = %q, want packages.go:199", r.Caller)
	}
	if r.Processor != "packages" || r.Provider != "brew" {
		t.Fatalf("attribution = %s/%s, want packages/brew", r.Processor, r.Provider)
	}
}

// A non-JSON line (a stray print) is kept verbatim, not dropped.
func TestJSONCaptureWriter_KeepsNonJSON(t *testing.T) {
	store := NewStore(100)
	w := &JSONCaptureWriter{Store: store}
	if _, err := w.Write([]byte("plain text line\n")); err != nil {
		t.Fatal(err)
	}
	records := store.Records(store.NewView(""))
	if len(records) != 1 || records[0].Msg != "plain text line" {
		t.Fatalf("records = %+v", records)
	}
}

// A message carrying embedded newlines (an error with a command's stderr
// blob) splits into one record per line: the display budgets panel height by
// record, and a single 20-line record pushed the whole frame past the
// terminal - the dashboard scrolled away and looked broken.
func TestAppend_SplitsMultilineMessages(t *testing.T) {
	store := NewStore(100)
	store.Append(LogRecord{Msg: "Error running command: exit status 101\nStderr: Updating index\n\n Downloading crates ...", Level: "ERRO"})
	records := store.Records(store.NewView(""))
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3 (blank line dropped): %+v", len(records), records)
	}
	for _, r := range records {
		if strings.Contains(r.Msg, "\n") {
			t.Fatalf("record still carries a newline: %q", r.Msg)
		}
		if r.Level != "ERRO" {
			t.Fatalf("split record lost its level: %+v", r)
		}
	}
}
