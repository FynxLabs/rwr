package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain strips styling so assertions see the text a human reads.
func plain(s string) string { return ansiSeq.ReplaceAllString(s, "") }

func testModel(t *testing.T) *Model {
	t.Helper()
	plan := &types.Plan{
		Init:  &types.InitConfig{},
		Order: []string{"packages", "files"},
		Resources: []types.Resource{
			{Processor: "packages", Provider: "pacman", Name: "git", Action: "install", Status: types.StatusOK},
			{Processor: "files", Name: "rc", Action: "create", Status: types.StatusFailed},
		},
	}
	store := reporting.NewStore(100)
	model := New(mustTheme("rwr"), plan, store, false, "")
	model.width, model.height = 100, 30
	return model
}

// Every frame renders from one model: running (with a failure row), summary,
// dry-run vocabulary, compact - and the ASCII theme - without panicking and
// with the load-bearing content present.
func TestModel_FramesRender(t *testing.T) {
	m := testModel(t)

	m.apply(reporting.ProcStarted{Processor: "packages"})
	m.apply(reporting.LaneUpdate{Processor: "packages", Provider: "pacman", Done: 3, Total: 10, Status: types.StatusOK})
	m.apply(reporting.ProcFinished{Processor: "packages", Dur: 2 * time.Second})
	m.apply(reporting.ProcStarted{Processor: "files"})
	m.apply(reporting.ProcFinished{Processor: "files", Err: errors.New("boom")})

	running := plain(m.render())
	for _, want := range []string{"rwr", "packages", "boom"} {
		if !strings.Contains(running, want) {
			t.Errorf("running frame missing %q:\n%s", want, running)
		}
	}

	m.apply(reporting.RunFinished{Errs: []types.StepError{{Processor: "files", Err: errors.New("boom")}}})
	summary := plain(m.render())
	for _, want := range []string{"failures", "boom", "applied"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary frame missing %q:\n%s", want, summary)
		}
	}

	m.dryRun = true
	if dry := plain(m.render()); !strings.Contains(dry, "DRY RUN") || !strings.Contains(dry, "would apply") {
		t.Errorf("dry-run frame missing badge/vocabulary:\n%s", dry)
	}

	m.compact = true
	if compact := plain(m.render()); compact == "" || strings.Count(compact, "\n") > m.height {
		t.Errorf("compact frame wrong shape:\n%s", compact)
	}
}

func TestResourceDoneMatchesSameNamedFilesByLocation(t *testing.T) {
	plan := &types.Plan{
		Init:  &types.InitConfig{},
		Order: []string{types.BlueprintTypeFiles},
		Resources: []types.Resource{
			{Processor: types.BlueprintTypeFiles, Name: "init.lua", Location: "/one/init.lua", Status: types.StatusPlanned},
			{Processor: types.BlueprintTypeFiles, Name: "init.lua", Location: "/two/init.lua", Status: types.StatusPlanned},
		},
	}
	m := New(mustTheme("rwr"), plan, reporting.NewStore(10), false, "")
	m.apply(reporting.ResourceDone{Resource: types.Resource{
		Processor: types.BlueprintTypeFiles,
		Name:      "init.lua",
		Location:  "/two/init.lua",
		Status:    types.StatusOK,
	}})

	if plan.Resources[0].Status != types.StatusPlanned {
		t.Errorf("first same-named file status = %s, want planned", plan.Resources[0].Status)
	}
	if plan.Resources[1].Status != types.StatusOK {
		t.Errorf("second same-named file status = %s, want ok", plan.Resources[1].Status)
	}
}

// Resize below either threshold trips compact mode live and back.
func TestModel_CompactModeOnResize(t *testing.T) {
	m := testModel(t)
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 30})
	if !m.compact {
		t.Error("50 cols did not trip compact mode")
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.compact {
		t.Error("resize back did not leave compact mode")
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})
	if !m.compact {
		t.Error("10 rows did not trip compact mode")
	}
}

// First manual input pins the selection; g unpins and snaps to live.
func TestModel_PinningFollowsDesign(t *testing.T) {
	m := testModel(t)
	m.apply(reporting.ProcStarted{Processor: "packages"})
	if m.pinned || m.cursor != 0 {
		t.Fatalf("selection did not follow execution: pinned=%v cursor=%d", m.pinned, m.cursor)
	}
	m.key(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if !m.pinned {
		t.Error("manual input did not pin")
	}
	m.apply(reporting.ProcStarted{Processor: "files"})
	if m.cursor != 0 {
		t.Errorf("pinned viewport moved with execution: cursor=%d", m.cursor)
	}
	m.key(tea.KeyPressMsg{Code: 'g', Text: "g"})
	if m.pinned || m.cursor != m.live {
		t.Error("g did not unpin and snap to live")
	}
}

// The ASCII theme renders every frame without unicode glyphs.
func TestModel_ASCIITheme(t *testing.T) {
	m := testModel(t)
	m.theme = mustThemeASCII("rwr")
	m.apply(reporting.ProcFinished{Processor: "packages", Dur: time.Second})
	out := plain(m.render())
	for _, glyph := range []string{"✓", "✗", "▰"} {
		if strings.Contains(out, glyph) {
			t.Errorf("ASCII theme leaked unicode glyph %q", glyph)
		}
	}
}
