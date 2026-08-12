package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

// The processor checklist is visible by default during a run: one named row
// per processor, whatever its state. The collapsed (x) mode remains the
// space-saving option; it must not be the default - a run whose progress list
// only appears on hover reads as frozen.
func TestViewRunning_ChecklistVisibleByDefault(t *testing.T) {
	plan := &types.Plan{Order: []string{"repositories", "packages", "scripts"}}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(100), false, "")
	m.width, m.height = 100, 30

	if !m.expanded {
		t.Fatal("checklist must be expanded by default")
	}

	m.procs[0].State = ProcDone
	m.procs[0].Dur = 2 * time.Second
	m.procs[1].State = ProcRunning
	m.procs[1].Started = time.Now()
	// procs[2] stays pending
	m.state = Running

	out := m.viewRunning()
	for _, name := range []string{"repositories", "packages", "scripts"} {
		if !strings.Contains(out, name) {
			t.Fatalf("running view missing checklist row for %q:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "pending") {
		t.Fatalf("pending processor not labeled in checklist:\n%s", out)
	}
}

// An interactive halt enters Prompting, and r/s/q answer the executor's
// decision channel: retry, skip, abort. The keys exist only at a halt.
func TestPrompting_HaltDecisions(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want reporting.HaltDecision
	}{
		{"r", reporting.HaltRetry},
		{"R", reporting.HaltRetry},
		{"s", reporting.HaltSkip},
		{"q", reporting.HaltAbort},
	} {
		plan := &types.Plan{Order: []string{"packages"}}
		m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
		m.width, m.height = 100, 30

		decision := make(chan reporting.HaltDecision, 1)
		m.apply(reporting.HaltReq{Processor: "packages", Err: errors.New("boom"), Decision: decision})
		if m.state != Prompting {
			t.Fatalf("state = %v after HaltReq, want Prompting", m.state)
		}

		// The typeahead grace window swallows keys right after the prompt
		// appears (leftover child-process input); age past it.
		m.resumedAt = time.Now().Add(-time.Second)

		m.key(tea.KeyPressMsg{Code: rune(tc.key[0]), Text: tc.key})
		select {
		case got := <-decision:
			if got != tc.want {
				t.Fatalf("key %q sent %v, want %v", tc.key, got, tc.want)
			}
		default:
			t.Fatalf("key %q sent no decision", tc.key)
		}
		if m.state != Running {
			t.Fatalf("state = %v after decision, want Running", m.state)
		}
	}
}

// Fresh-machine ghost lane: stage 2 planned unpinned entries into an "items"
// lane because no provider existed yet; the first runtime LaneUpdate for the
// resolved provider re-keys that lane instead of leaving it pending forever
// next to the real one.
func TestLaneUpdate_RekeysGhostItemsLane(t *testing.T) {
	plan := &types.Plan{
		Order: []string{"packages"},
		Resources: []types.Resource{
			{Processor: "packages", Provider: "", Name: "jq", Action: "install", Status: types.StatusPlanned},
			{Processor: "packages", Provider: "", Name: "fd", Action: "install", Status: types.StatusPlanned},
		},
	}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")

	if _, has := m.procs[0].Lanes["items"]; !has {
		t.Fatal("plan lane for unpinned entries missing")
	}

	m.apply(reporting.LaneUpdate{Processor: "packages", Provider: "brew", Done: 1, Total: 2, Status: types.StatusOK})

	if _, has := m.procs[0].Lanes["items"]; has {
		t.Fatal("ghost items lane survived the first provider update")
	}
	lane, has := m.procs[0].Lanes["brew"]
	if !has {
		t.Fatal("re-keyed brew lane missing")
	}
	if lane.Done != 1 || lane.Total != 2 {
		t.Fatalf("re-keyed lane counts = %d/%d, want 1/2", lane.Done, lane.Total)
	}
}
