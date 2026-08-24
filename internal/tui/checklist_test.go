package tui

import (
	"errors"
	"strings"
	"sync/atomic"
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
	t.Parallel()

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
		m.apply(reporting.HaltReq{Processor: "packages", Err: errors.New("boom"), Retryable: true, Decision: decision})
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

func TestPrompting_AllowsMouseRelease(t *testing.T) {
	t.Parallel()

	plan := &types.Plan{Order: []string{"users"}}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	m.width, m.height = 100, 30
	m.apply(reporting.HaltReq{Processor: "users", Err: errors.New("boom"), Retryable: true, Decision: make(chan reporting.HaltDecision, 1)})
	m.resumedAt = time.Now().Add(-time.Second)

	m.key(tea.KeyPressMsg{Code: 'm', Text: "m"})
	if m.mouseCapture {
		t.Fatal("m did not release mouse capture while the halt prompt was active")
	}
	if m.state != Prompting || m.halt == nil {
		t.Fatal("mouse toggle answered or dismissed the halt prompt")
	}
}

func TestPrompting_FinalFailuresRequireAcknowledgement(t *testing.T) {
	t.Parallel()

	plan := &types.Plan{Order: []string{"run"}}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	m.width, m.height = 100, 30
	decision := make(chan reporting.HaltDecision, 1)
	m.apply(reporting.HaltReq{Processor: "run", Err: errors.New("operation failed"), Decision: decision})
	m.resumedAt = time.Now().Add(-time.Second)

	if view := m.View().Content; strings.Contains(view, "retry") {
		t.Fatalf("final failure prompt offered unsafe retry:\n%s", view)
	}
	m.key(tea.KeyPressMsg{Code: 'r', Text: "r"})
	select {
	case got := <-decision:
		t.Fatalf("retry key answered non-retryable final prompt with %v", got)
	default:
	}
	if m.state != Prompting {
		t.Fatalf("retry key dismissed final prompt; state = %v", m.state)
	}

	m.key(tea.KeyPressMsg{Code: 's', Text: "s"})
	if got := <-decision; got != reporting.HaltSkip {
		t.Fatalf("acknowledge sent %v, want %v", got, reporting.HaltSkip)
	}
}

func TestPrompting_SecretStaysInsideTUIAndIsMasked(t *testing.T) {
	t.Parallel()

	plan := &types.Plan{Order: []string{"users"}}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	m.width, m.height = 100, 30
	result := make(chan reporting.SecretResult, 1)
	m.apply(reporting.SecretReq{Prompt: "sudo password", Result: result})

	for _, r := range "s3cret" {
		m.key(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	rendered := m.View()
	view := rendered.Content
	if strings.Contains(view, "s3cret") {
		t.Fatal("secret was rendered in the TUI")
	}
	if !strings.Contains(view, "••••••") {
		t.Fatalf("masked input not rendered:\n%s", view)
	}
	if !strings.Contains(view, "ADMINISTRATOR AUTHENTICATION REQUIRED") {
		t.Fatalf("password modal title not rendered:\n%s", view)
	}
	if rendered.Cursor == nil || rendered.Cursor.Shape != tea.CursorBar {
		t.Fatalf("password dialog has no real bar cursor: %#v", rendered.Cursor)
	}

	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	answer := <-result
	if answer.Err != nil || string(answer.Value) != "s3cret" {
		t.Fatalf("secret result = (%q, %v)", answer.Value, answer.Err)
	}
	for i := range answer.Value {
		answer.Value[i] = 0
	}
	if m.state != Running || m.secret != nil {
		t.Fatal("secret prompt did not return the model to running")
	}
}

func TestPrompting_LostSecretClaimStillClearsModelInput(t *testing.T) {
	t.Parallel()

	var claimed atomic.Bool
	claimed.Store(true)
	m := New(mustTheme("rwr"), &types.Plan{Order: []string{"users"}}, reporting.NewStore(10), false, "")
	m.apply(reporting.SecretReq{Prompt: "sudo password", Result: make(chan reporting.SecretResult, 1), Claim: &claimed})
	for _, r := range "secret" {
		m.key(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.secretValue != nil || m.secret != nil || m.state != Running {
		t.Fatal("lost secret claim did not clear prompt state")
	}
}

func TestSanitizePromptText_RemovesTerminalControls(t *testing.T) {
	t.Parallel()

	input := "Overwrite /tmp/ok\x1b]52;c;c3RvbGVu\a\r.conf"
	if got, want := sanitizePromptText(input), "Overwrite /tmp/ok.conf"; got != want {
		t.Fatalf("sanitizePromptText() = %q, want %q", got, want)
	}
}

func TestPrompting_ReengagesLiveProcessorFollow(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		event reporting.Event
	}{
		{name: "secret", event: reporting.SecretReq{Prompt: "sudo password", Result: make(chan reporting.SecretResult, 1)}},
		{name: "confirmation", event: reporting.ConfirmReq{Prompt: "overwrite?", Result: make(chan reporting.ConfirmResult, 1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := &types.Plan{Order: []string{"packages", "users"}}
			m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
			m.width, m.height = 100, 30
			m.apply(reporting.ProcStarted{Processor: "packages"})
			m.cursor, m.pinned, m.scrollOffset = 0, true, 12
			m.apply(reporting.ProcFinished{Processor: "packages"})
			m.apply(reporting.ProcStarted{Processor: "users"})
			if m.cursor != 0 {
				t.Fatal("test setup did not preserve the pinned packages viewport")
			}

			m.apply(tc.event)
			if m.pinned || m.cursor != 1 || m.scrollOffset != 0 {
				t.Fatalf("prompt did not resume live follow: pinned=%v cursor=%d scroll=%d", m.pinned, m.cursor, m.scrollOffset)
			}
		})
	}
}

func TestPrompting_ConfirmationStaysInsideTUI(t *testing.T) {
	t.Parallel()

	plan := &types.Plan{Order: []string{"files"}}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	m.width, m.height = 100, 30
	result := make(chan reporting.ConfirmResult, 1)
	m.apply(reporting.ConfirmReq{Prompt: "Overwrite existing file? /tmp/config", AllowAll: true, Result: result})

	if view := m.View().Content; !strings.Contains(view, "Overwrite existing file?") || !strings.Contains(view, "ACTION REQUIRED") || !strings.Contains(strings.ToLower(view), "overwrite all") {
		t.Fatalf("confirmation modal not rendered with all option:\n%s", view)
	}
	m.key(tea.KeyPressMsg{Code: 'a', Text: "a"})
	answer := <-result
	if answer.Err != nil || !answer.Yes || !answer.All {
		t.Fatalf("confirmation result = (yes=%v, all=%v, err=%v)", answer.Yes, answer.All, answer.Err)
	}
	if m.state != Running || m.confirm != nil {
		t.Fatal("confirmation did not return the model to running")
	}
}

func TestPrompting_ConfirmationButtonsDefaultSafeAndNavigate(t *testing.T) {
	t.Parallel()

	newDialog := func() (*Model, chan reporting.ConfirmResult) {
		result := make(chan reporting.ConfirmResult, 1)
		m := New(mustTheme("rwr"), &types.Plan{Order: []string{"files"}}, reporting.NewStore(10), false, "")
		m.width, m.height = 100, 30
		m.apply(reporting.ConfirmReq{Prompt: "Overwrite?", AllowAll: true, Result: result})
		return m, result
	}

	m, result := newDialog()
	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	if answer := <-result; answer.Yes || answer.All || answer.Err != nil {
		t.Fatalf("default button was not safe Skip: %+v", answer)
	}

	m, result = newDialog()
	m.key(tea.KeyPressMsg{Code: tea.KeyRight}) // Skip -> Skip All.
	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	if answer := <-result; answer.Yes || !answer.All || answer.Err != nil {
		t.Fatalf("Skip All button did not return a persistent skip: %+v", answer)
	}

	m, result = newDialog()
	m.key(tea.KeyPressMsg{Code: tea.KeyRight}) // Skip -> Skip All.
	m.key(tea.KeyPressMsg{Code: tea.KeyRight}) // Skip All wraps to Overwrite.
	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	if answer := <-result; !answer.Yes || answer.All || answer.Err != nil {
		t.Fatalf("button navigation did not select Overwrite: %+v", answer)
	}

	m, result = newDialog()
	m.key(tea.KeyPressMsg{Code: 's', Text: "s"})
	if answer := <-result; answer.Yes || !answer.All || answer.Err != nil {
		t.Fatalf("skip-all shortcut returned wrong answer: %+v", answer)
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

// The mixed fresh-Mac layout: unpinned entries (default provider, planned as
// "items" because no provider existed at stage 2) alongside entries pinned to
// cargo. The first brew update must re-key the ghost even though the
// processor already has a cargo lane.
func TestLaneUpdate_RekeysGhostAmongPinnedLanes(t *testing.T) {
	plan := &types.Plan{
		Order: []string{"packages"},
		Resources: []types.Resource{
			{Processor: "packages", Provider: "", Name: "jq", Action: "install", Status: types.StatusPlanned},
			{Processor: "packages", Provider: "cargo", Name: "eza", Action: "install", Status: types.StatusPlanned},
		},
	}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	if len(m.procs[0].Lanes) != 2 {
		t.Fatalf("plan lanes = %d, want 2 (items + cargo)", len(m.procs[0].Lanes))
	}

	m.apply(reporting.LaneUpdate{Processor: "packages", Provider: "brew", Done: 1, Total: 1, Status: types.StatusOK})

	if _, has := m.procs[0].Lanes["items"]; has {
		t.Fatal("ghost items lane survived despite the pinned cargo lane")
	}
	if _, has := m.procs[0].Lanes["brew"]; !has {
		t.Fatal("brew lane missing after re-key")
	}
	if _, has := m.procs[0].Lanes["cargo"]; !has {
		t.Fatal("cargo lane lost")
	}
}
