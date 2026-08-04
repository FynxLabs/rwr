package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// View implements tea.Model. Both layouts render from the same model, so a
// live resize promotes or demotes with no state lost.
func (m *Model) View() tea.View {
	// Scan registers the zone marks the renderers injected and strips their
	// escape sequences; all-motion mouse mode is what delivers hover events.
	view := tea.NewView(m.zones.Scan(m.render()))
	view.MouseMode = tea.MouseModeAllMotion
	return view
}

func (m *Model) render() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.compact {
		return m.viewCompact()
	}
	switch m.state {
	case Resolving:
		return m.viewResolving()
	case SummaryState:
		return m.viewSummary()
	default:
		return m.viewRunning()
	}
}

func (m *Model) glyphFor(state ProcState) (string, lipgloss.Style) {
	g := m.theme.Glyphs
	switch state {
	case ProcDone:
		return g.Done, style(m.theme.Success)
	case ProcFailed:
		return g.Failed, style(m.theme.Danger)
	case ProcDegraded:
		return g.Degraded, style(m.theme.Warning)
	case ProcSkipped:
		return g.Skipped, style(m.theme.Muted)
	case ProcRunning:
		return g.Spinner[m.spinnerFrame%len(g.Spinner)], style(m.theme.Accent)
	default:
		return g.Pending, style(m.theme.Dim)
	}
}

// header: name, source, failure counters, elapsed clock. Fixed, never
// sacrificed.
func (m *Model) viewHeader() string {
	elapsed := time.Since(m.started).Round(time.Second)
	if !m.finished.IsZero() {
		elapsed = m.finished.Sub(m.started).Round(time.Second)
	}
	failures := 0
	for _, proc := range m.procs {
		if proc.State == ProcFailed || proc.State == ProcDegraded {
			failures++
		}
	}
	left := style(m.theme.Accent).Bold(true).Render("rwr")
	if m.dryRun {
		left += " " + style(m.theme.Warning).Bold(true).Render("DRY RUN")
	}
	if m.plan != nil && m.plan.Init != nil {
		left += " " + style(m.theme.Subtext).Render(m.plan.Init.Init.Location)
	}
	right := ""
	if failures > 0 {
		right += style(m.theme.Danger).Render(fmt.Sprintf("%s %d ", m.theme.Glyphs.Failed, failures))
	}
	right += style(m.theme.Muted).Render(elapsed.String())
	pad := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

// strip: one block per processor, colored by state, one line total. Each
// cell is a mouse zone; the hovered cell renders reversed so the pointer's
// target reads before the click.
func (m *Model) viewStrip() string {
	var b strings.Builder
	for i, proc := range m.procs {
		glyph := m.theme.Glyphs.StripFilled
		if m.dryRun || proc.State == ProcPending {
			glyph = m.theme.Glyphs.StripEmpty
		}
		_, st := m.glyphFor(proc.State)
		if i == m.hovered {
			st = st.Reverse(true)
		}
		b.WriteString(m.zones.Mark(stripZone(i), st.Render(glyph)))
	}
	if m.hovered >= 0 && m.hovered < len(m.procs) {
		b.WriteString(" " + style(m.theme.Subtext).Render(m.procs[m.hovered].Name))
	}
	return b.String()
}

// collapsed: successful processors compress onto one shared line; failed and
// degraded always keep a full row with the reason in place of a duration.
func (m *Model) viewCollapsed() []string {
	var lines []string
	var done []string
	var doneDur time.Duration
	for i, proc := range m.procs {
		nameStyle := style(m.theme.Text)
		if i == m.hovered {
			nameStyle = nameStyle.Underline(true)
		}
		switch proc.State {
		case ProcDone:
			if m.expanded {
				glyph, st := m.glyphFor(proc.State)
				lines = append(lines, m.zones.Mark(rowZone(i), fmt.Sprintf(" %s %s %s", st.Render(glyph), nameStyle.Render(fmt.Sprintf("%-16s", proc.Name)), style(m.theme.Muted).Render(proc.Dur.Round(time.Millisecond*100).String()))))
			} else {
				done = append(done, proc.Name)
				doneDur += proc.Dur
			}
		case ProcFailed, ProcDegraded:
			glyph, st := m.glyphFor(proc.State)
			reason := ""
			if proc.Err != nil {
				reason = proc.Err.Error()
			}
			lines = append(lines, m.zones.Mark(rowZone(i), fmt.Sprintf(" %s %s %s", st.Render(glyph), nameStyle.Render(fmt.Sprintf("%-16s", proc.Name)), style(m.theme.Danger).Render(truncate(reason, m.width-24)))))
		}
	}
	if len(done) > 0 && !m.expanded {
		line := fmt.Sprintf(" %s %s  %s",
			style(m.theme.Success).Render(m.theme.Glyphs.Done),
			style(m.theme.Subtext).Render(strings.Join(done, " · ")),
			style(m.theme.Muted).Render(doneDur.Round(time.Millisecond*100).String()))
		lines = append([]string{truncate(line, m.width)}, lines...)
	}
	return lines
}

// panel: the active processor with provider lanes and its log viewport. The
// border takes the worst lane state.
func (m *Model) viewPanel(height int) string {
	if m.cursor >= len(m.procs) || height < 3 {
		return ""
	}
	proc := m.procs[m.cursor]
	_, borderStyle := m.glyphFor(proc.State)

	var b strings.Builder
	for _, lane := range proc.Lanes {
		bar := renderFillBar(lane.fill, 20, m.theme.Glyphs)
		// Denominators are declared resources, not operations: 37/112 with
		// the log carrying the rest, never implying the count is total work.
		fmt.Fprintf(&b, " %-12s %s %d/%d\n", lane.Provider, bar, lane.Done, lane.Total)
	}

	// Log viewport: the per-processor view, with global filters on top.
	logLines := m.filteredLines(height - len(proc.Lanes) - 2)
	for _, line := range logLines {
		b.WriteString(" " + truncate(line, m.width-4) + "\n")
	}

	border := lipgloss.RoundedBorder()
	if m.theme.Glyphs.Done == "+" { // ASCII theme
		border = lipgloss.ASCIIBorder()
	}
	panel := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderStyle.GetForeground()).
		Width(m.width - 2).
		Render(strings.TrimRight(b.String(), "\n"))
	return panel
}

// filteredLines applies the global level/stdout/search filters to a view.
func (m *Model) filteredLines(max int) []string {
	if max < 3 {
		max = 3 // active panel minimum, per the space priority
	}
	records := m.records()
	var lines []string
	for _, record := range records {
		if m.levelFilter != "" && record.Level != m.levelFilter {
			continue
		}
		if m.stdoutFilter && record.Src == 0 {
			continue
		}
		if m.search != "" && !strings.Contains(strings.ToLower(record.Msg), strings.ToLower(m.search)) {
			continue
		}
		lines = append(lines, record.Msg)
	}
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return lines
}

// help bar; collapses to `? help` when space is short.
func (m *Model) viewHelp(short bool) string {
	if short {
		return style(m.theme.Dim).Render(" ? help")
	}
	keys := " j/k select · g follow · e errors · o stdout · / search · x expand · q quit"
	if m.searching {
		keys = " search: " + m.search + "▌"
	}
	return style(m.theme.Dim).Render(truncate(keys, m.width))
}

func (m *Model) viewResolving() string {
	// Held behind a delay by the runner; when shown: the stage list.
	spin := m.theme.Glyphs.Spinner[m.spinnerFrame%len(m.theme.Glyphs.Spinner)]
	var b strings.Builder
	b.WriteString(m.viewHeader() + "\n\n")
	b.WriteString(" " + style(m.theme.Accent).Render(spin) + " resolving blueprints…\n\n")
	b.WriteString(style(m.theme.Muted).Render(" nothing has been modified yet - q cancels") + "\n")
	return b.String()
}

func (m *Model) viewRunning() string {
	var b strings.Builder
	b.WriteString(m.viewHeader() + "\n")
	b.WriteString(m.viewStrip() + "\n")
	collapsed := m.viewCollapsed()
	for _, line := range collapsed {
		b.WriteString(line + "\n")
	}
	panelHeight := m.height - 4 - len(collapsed)
	b.WriteString(m.viewPanel(panelHeight) + "\n")
	// pending chips
	var pending []string
	for _, proc := range m.procs {
		if proc.State == ProcPending {
			pending = append(pending, proc.Name)
		}
	}
	if len(pending) > 0 {
		b.WriteString(style(m.theme.Dim).Render(truncate(" pending: "+strings.Join(pending, " "), m.width)) + "\n")
	}
	b.WriteString(m.viewHelp(m.height < 20))
	return b.String()
}

// viewSummary: tabs per processor plus a failures tab, first and selected by
// default; dry-run renders the same frame with its own vocabulary.
func (m *Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(m.viewHeader() + "\n")
	b.WriteString(m.viewStrip() + "\n\n")

	tabs := append([]string{"failures"}, m.plan.Order...)
	var tabLine []string
	for i, tab := range tabs {
		if i == m.summaryTab {
			tabLine = append(tabLine, style(m.theme.Accent).Bold(true).Underline(true).Render(tab))
		} else {
			tabLine = append(tabLine, style(m.theme.Muted).Render(tab))
		}
	}
	b.WriteString(" " + truncate(strings.Join(tabLine, "  "), m.width) + "\n\n")

	applied, skipped, failed, unknown := 0, 0, 0, 0
	for _, res := range m.plan.Resources {
		switch res.Status {
		case types.StatusOK, types.StatusPlanned:
			applied++
		case types.StatusSkipped, types.StatusPresent:
			skipped++
		case types.StatusFailed:
			failed++
		default:
			unknown++
		}
	}

	if m.summaryTab == 0 {
		if len(m.errs) == 0 {
			b.WriteString(style(m.theme.Success).Render(" no failures") + "\n")
		}
		for _, stepErr := range m.errs {
			fmt.Fprintf(&b, " %s %-14s %s\n",
				style(m.theme.Danger).Render(m.theme.Glyphs.Failed),
				stepErr.Processor,
				truncate(stepErr.Err.Error(), m.width-20))
		}
	} else {
		processor := tabs[m.summaryTab]
		count := 0
		for _, res := range m.plan.Resources {
			if res.Processor != processor {
				continue
			}
			if m.search != "" && !strings.Contains(res.Name, m.search) {
				continue
			}
			glyph := m.statusGlyph(res.Status)
			fmt.Fprintf(&b, " %s %-10s %-24s %s\n", glyph, res.Provider, truncate(res.Name, 24), res.Action)
			count++
			if count > m.height-10 {
				b.WriteString(style(m.theme.Dim).Render("  …") + "\n")
				break
			}
		}
	}

	b.WriteString("\n")
	if m.dryRun {
		b.WriteString(style(m.theme.Muted).Render(fmt.Sprintf(" would apply %d · already present %d · unresolved %d", applied, skipped, unknown)) + "\n")
	} else {
		b.WriteString(style(m.theme.Muted).Render(fmt.Sprintf(" applied %d · skipped %d · failed %d · unknown %d", applied, skipped, failed, unknown)) + "\n")
	}
	if m.runLogPath != "" {
		b.WriteString(style(m.theme.Dim).Render(" run log: "+m.runLogPath) + "\n")
	}
	b.WriteString(m.viewHelp(false))
	return b.String()
}

func (m *Model) statusGlyph(status types.Status) string {
	g := m.theme.Glyphs
	if m.dryRun {
		// Dry-run vocabulary: + would apply, = already present, ? cannot
		// resolve. Amber, not red: a plan problem, not a failure.
		switch status {
		case types.StatusPresent:
			return style(m.theme.Muted).Render("=")
		case types.StatusUnknown:
			return style(m.theme.Warning).Render("?")
		default:
			return style(m.theme.Success).Render("+")
		}
	}
	switch status {
	case types.StatusOK:
		return style(m.theme.Success).Render(g.Done)
	case types.StatusFailed:
		return style(m.theme.Danger).Render(g.Failed)
	case types.StatusSkipped:
		return style(m.theme.Muted).Render(g.Skipped)
	default:
		return style(m.theme.Warning).Render(g.Unknown)
	}
}

// viewCompact: one pinned status line, the log stream, one help line - stays
// inside the TUI so filters and search survive a tiny pane.
func (m *Model) viewCompact() string {
	failures := 0
	for _, proc := range m.procs {
		if proc.State == ProcFailed {
			failures++
		}
	}
	running := ""
	if m.live < len(m.procs) {
		running = m.procs[m.live].Name
	}
	var b strings.Builder
	statusLine := fmt.Sprintf("rwr %s %d/%d", running, m.live+1, len(m.procs))
	if failures > 0 {
		statusLine += style(m.theme.Danger).Render(fmt.Sprintf(" %s%d", m.theme.Glyphs.Failed, failures))
	}
	b.WriteString(truncate(statusLine, m.width) + "\n")
	for _, line := range m.filteredLines(m.height - 2) {
		b.WriteString(truncate(line, m.width) + "\n")
	}
	b.WriteString(m.viewHelp(true))
	return b.String()
}

func (m *Model) records() []recordView {
	view := m.views[m.scope]
	if view == nil {
		view = m.views[""]
	}
	records := m.store.Records(view)
	out := make([]recordView, 0, len(records))
	for _, record := range records {
		out = append(out, recordView{Msg: record.Msg, Level: record.Level, Src: int(record.Src)})
	}
	return out
}

type recordView struct {
	Msg   string
	Level string
	Src   int
}

func renderFillBar(fraction float64, width int, glyphs Glyphs) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(fraction * float64(width))
	return strings.Repeat(glyphs.BarFull, filled) + strings.Repeat(glyphs.BarEmpty, width-filled)
}

func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + "…"
}
