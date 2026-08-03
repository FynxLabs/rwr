package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// stripCellZone waits for the zone scan to register the first strip cell;
// the zone manager parses marks on Scan but records them asynchronously.
func stripCellZone(t *testing.T, m *Model) (x, y int) {
	t.Helper()
	m.state = Running // the resolving frame has no strip
	m.View()          // render once so Scan registers the marks
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if info := m.zones.Get(stripZone(0)); info != nil && !info.IsZero() {
			return info.StartX, info.StartY
		}
		time.Sleep(5 * time.Millisecond)
		m.View()
	}
	t.Fatal("strip zone never registered")
	return 0, 0
}

func TestMouseHoverHighlightsStripCell(t *testing.T) {
	m := testModel(t)
	x, y := stripCellZone(t, m)

	m.Update(tea.MouseMotionMsg{X: x, Y: y})
	if m.hovered != 0 {
		t.Fatalf("hovered = %d after motion over the first strip cell, want 0", m.hovered)
	}
	if got := plain(m.render()); !strings.Contains(got, "packages") {
		t.Fatalf("hover hint missing from render:\n%s", got)
	}

	m.Update(tea.MouseMotionMsg{X: x, Y: y + 20})
	if m.hovered != -1 {
		t.Fatalf("hovered = %d after motion away, want -1", m.hovered)
	}
}

func TestMouseClickSelectsAndPins(t *testing.T) {
	m := testModel(t)
	x, y := stripCellZone(t, m)
	m.cursor = 1

	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if !m.pinned {
		t.Fatal("click did not pin")
	}
	if m.cursor != 0 {
		t.Fatalf("cursor = %d after click on the first strip cell, want 0", m.cursor)
	}
}
