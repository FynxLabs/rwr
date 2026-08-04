package reporting

import (
	"fmt"
	"testing"
)

// Every line written while a processor is stamped is attributed to it - the
// mechanism that gives per-processor log views with zero edits to the ten
// processor files.
func TestLineWriter_AttributesRecordsToCurrentProcessor(t *testing.T) {
	store := NewStore(100)
	writer := &LineWriter{Store: store, Src: SrcLog}

	SetCurrentProcessor("packages")
	fmt.Fprintf(writer, "INFO Installing git\npartial")
	SetCurrentProcessor("files")
	fmt.Fprintf(writer, " line finished\nINFO Copying rc\n")

	view := store.NewView("")
	records := store.Records(view)
	if len(records) != 3 {
		t.Fatalf("records = %d, want 3: %+v", len(records), records)
	}
	if records[0].Processor != "packages" || records[0].Level != "INFO" {
		t.Errorf("record 0 = %+v, want packages/INFO", records[0])
	}
	// The partial line completed after the stamp changed: it belongs to the
	// processor that finished it.
	if records[1].Processor != "files" || records[1].Msg != "partial line finished" {
		t.Errorf("record 1 = %+v, want files owning the completed partial line", records[1])
	}

	packagesView := store.NewView("packages")
	if got := store.Records(packagesView); len(got) != 1 {
		t.Errorf("packages view = %d records, want 1: %+v", len(got), got)
	}
}

// Views hold Seqs, not indices: once the ring wraps, stale Seqs drop lazily
// and the survivors still resolve to the right records.
func TestStore_RingEvictionKeepsSeqsStable(t *testing.T) {
	store := NewStore(4)
	view := store.NewView("")
	SetCurrentProcessor("packages")

	for i := 1; i <= 10; i++ {
		store.Append(LogRecord{Msg: fmt.Sprintf("line %d", i), Processor: "packages"})
	}

	records := store.Records(view)
	if len(records) != 4 {
		t.Fatalf("records = %d, want the 4 ring survivors", len(records))
	}
	if records[0].Msg != "line 7" || records[3].Msg != "line 10" {
		t.Errorf("survivors = %q..%q, want line 7..line 10", records[0].Msg, records[3].Msg)
	}
	if store.Oldest() != records[0].Seq {
		t.Errorf("Oldest = %d, want %d", store.Oldest(), records[0].Seq)
	}
}
