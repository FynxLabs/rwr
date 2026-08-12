package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

// View implements tea.Model. Both layouts render from the same model, so a
// live resize promotes or demotes with no state lost.
func (m *Model) View() tea.View {
	// Scan registers the zone marks the renderers injected and strips their
	// escape sequences; all-motion mouse mode is what delivers hover events.
	// m toggles capture off so terminal-native selection and copy work.
	view := tea.NewView(m.zones.Scan(m.render()))
	if m.mouseCapture {
		view.MouseMode = tea.MouseModeAllMotion
	} else {
		view.MouseMode = tea.MouseModeNone
	}
	// The dashboard is a full-frame app: the alternate screen buffer is what
	// keeps frames from smearing into the terminal's scrollback. Focus
	// reporting feeds the completion notification's unfocused gate.
	view.AltScreen = true
	view.ReportFocus = true
	view.WindowTitle = m.windowTitle()
	view.ProgressBar = m.taskbarProgress()
	return view
}

// windowTitle names the run in the terminal tab.
func (m *Model) windowTitle() string {
	failed := 0
	for _, proc := range m.procs {
		if proc.State == ProcFailed || proc.State == ProcDegraded {
			failed++
		}
	}
	switch {
	case m.state == SummaryState && failed > 0:
		return fmt.Sprintf("rwr · done · %d failed", failed)
	case m.state == SummaryState:
		return "rwr · done"
	default:
		return "rwr · running"
	}
}

// taskbarProgress emits OSC 9;4: overall run percent from finished
// processors, error state on the first failure. Passive state, not an alert -
// terminals that don't render it ignore it.
func (m *Model) taskbarProgress() *tea.ProgressBar {
	if len(m.procs) == 0 {
		return nil
	}
	finished, failed := 0, 0
	for _, proc := range m.procs {
		switch proc.State {
		case ProcDone, ProcSkipped:
			finished++
		case ProcFailed, ProcDegraded:
			finished++
			failed++
		}
	}
	if m.state == SummaryState {
		return nil // clear it when the run ends
	}
	state := tea.ProgressBarDefault
	if failed > 0 {
		state = tea.ProgressBarError
	}
	return tea.NewProgressBar(state, finished*100/len(m.procs))
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
		// Two cells plus separator: a 1-cell block is too small a mouse
		// target; the separator keeps neighbors from reading as one bar.
		b.WriteString(m.zones.Mark(stripZone(i), st.Render(glyph+glyph)) + " ")
	}
	if m.hovered >= 0 && m.hovered < len(m.procs) {
		b.WriteString(style(m.theme.Subtext).Render(m.procs[m.hovered].Name))
	}
	return b.String()
}

// viewCollapsed renders the processor checklist. Expanded (the default),
// every processor keeps its own named row - done with a duration, running
// with its elapsed clock, pending dimmed, failed with the reason - so the
// run's shape is always on screen without hovering anything. Collapsed
// (toggle with x), successful processors compress onto one shared line to
// give the log panel the vertical space; failed and degraded always keep a
// full row either way.
func (m *Model) viewCollapsed() []string {
	var lines []string
	var done []string
	var doneDur time.Duration
	for i, proc := range m.procs {
		nameStyle := style(m.theme.Text)
		if i == m.hovered {
			nameStyle = nameStyle.Underline(true)
		}
		glyph, st := m.glyphFor(proc.State)
		row := func(detail string) string {
			return m.zones.Mark(rowZone(i), fmt.Sprintf(" %s %s %s", st.Render(glyph), nameStyle.Render(fmt.Sprintf("%-16s", proc.Name)), detail))
		}
		switch proc.State {
		case ProcDone:
			if m.expanded {
				lines = append(lines, row(style(m.theme.Muted).Render(proc.Dur.Round(time.Millisecond*100).String())))
			} else {
				done = append(done, proc.Name)
				doneDur += proc.Dur
			}
		case ProcFailed, ProcDegraded:
			reason := ""
			if proc.Err != nil {
				// One line no matter what the error carries: an embedded
				// newline in a checklist row breaks the whole height budget.
				reason = strings.ReplaceAll(proc.Err.Error(), "\n", " · ")
			}
			lines = append(lines, row(style(m.theme.Danger).Render(truncate(reason, m.width-24))))
			// Failed subtrees never fold: the lanes stay visible so the
			// failing provider is identifiable at a glance.
			if m.expanded && len(proc.Lanes) > 0 {
				lines = append(lines, m.laneRows(proc)...)
			}
		case ProcRunning:
			if m.expanded {
				lines = append(lines, row(style(m.theme.Muted).Render(time.Since(proc.Started).Round(time.Second).String())))
				// Provider lanes nest under their processor's checklist row:
				//   ⠧ packages        12s
				//      brew   ███░░ 66/107
				//      cargo  ░░░░░ 0/19
				lines = append(lines, m.laneRows(proc)...)
			}
		case ProcSkipped:
			if m.expanded {
				lines = append(lines, row(style(m.theme.Muted).Render("skipped")))
			}
		case ProcPending:
			if m.expanded {
				lines = append(lines, row(style(m.theme.Dim).Render("pending")))
			}
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

// laneRows renders a processor's provider lanes as indented checklist rows,
// sorted so the map's random iteration cannot re-shuffle them every frame.
// Each child row carries its own glyph; a done lane keeps its full bar with ✓
// appended (per the normative frame). Denominators are declared resources,
// not operations: 37/112 with the log carrying the rest, never implying the
// count is total work.
func (m *Model) laneRows(proc *Proc) []string {
	laneNames := make([]string, 0, len(proc.Lanes))
	for name := range proc.Lanes {
		laneNames = append(laneNames, name)
	}
	sort.Strings(laneNames)
	rows := make([]string, 0, len(laneNames))
	for _, name := range laneNames {
		lane := proc.Lanes[name]
		bar := renderFillBar(lane.fill, 20, m.theme.Glyphs)
		g := m.theme.Glyphs
		var glyph, suffix string
		switch {
		case lane.Status == types.StatusFailed:
			glyph = style(m.theme.Danger).Render(g.Failed)
		case lane.Total > 0 && lane.Done >= lane.Total:
			glyph = style(m.theme.Success).Render(g.Done)
			suffix = " " + style(m.theme.Success).Render(g.Done)
		case proc.State == ProcRunning:
			glyph = style(m.theme.Accent).Render(g.Spinner[m.spinnerFrame%len(g.Spinner)])
		default:
			glyph = style(m.theme.Dim).Render(g.Pending)
		}
		rows = append(rows, fmt.Sprintf("    %s %-8s %s %d/%d%s", glyph, lane.Provider, bar, lane.Done, lane.Total, suffix))
	}
	return rows
}

// panel: the selected processor's log viewport. Its provider lanes live
// nested under the processor's checklist row, not in here; collapsed mode
// (which has no running rows to nest under) keeps them at the top of the
// panel so progress stays visible. The border takes the worst lane state.
func (m *Model) viewPanel(height int) string {
	if m.cursor >= len(m.procs) || height < 3 {
		return ""
	}
	proc := m.procs[m.cursor]
	_, borderStyle := m.glyphFor(proc.State)

	var b strings.Builder
	laneCount := 0
	if !m.expanded {
		laneRows := m.laneRows(proc)
		laneCount = len(laneRows)
		for _, row := range laneRows {
			b.WriteString(row + "\n")
		}
	}

	// Log viewport: the per-processor view, with global filters on top.
	logLines := m.filteredLines(height - laneCount - 2)
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

// visibleRecords applies the display level and output filters to the active
// view. Search highlights rather than filters, so it does not remove lines.
func (m *Model) visibleRecords() []recordView {
	records := m.records()
	out := records[:0:0]
	for _, record := range records {
		if m.recordVisible(record) {
			out = append(out, record)
		}
	}
	return out
}

// window slices the tail window of lines honoring the scroll offset, and
// clamps the offset so scrolling past the top sticks at the top.
func (m *Model) window(total, max int) (start, end int) {
	if total <= max {
		m.scrollOffset = 0
		return 0, total
	}
	if m.scrollOffset > total-max {
		m.scrollOffset = total - max
	}
	end = total - m.scrollOffset
	return end - max, end
}

// filteredLines renders the visible tail window of the active view.
func (m *Model) filteredLines(max int) []string {
	if max < 3 {
		max = 3 // active panel minimum, per the space priority
	}
	records := m.visibleRecords()
	start, end := m.window(len(records), max)
	lines := make([]string, 0, end-start)
	for _, record := range records[start:end] {
		lines = append(lines, m.renderRecord(record))
	}
	m.lastLogLines = max
	return lines
}

// plainLines is the same window without styling, for the y yank.
func (m *Model) plainLines(max int) []string {
	if max < 3 {
		max = 3
	}
	records := m.visibleRecords()
	start, end := m.window(len(records), max)
	lines := make([]string, 0, end-start)
	for _, record := range records[start:end] {
		line := record.Time.Format("15:04:05") + " "
		if record.Provider != "" {
			line += record.Provider + " "
		}
		line += normalizeMsg(record.Msg)
		lines = append(lines, line)
	}
	return lines
}

// panelLogLines is the log capacity of the last rendered panel; key handlers
// use it for page and resize step sizes.
func (m *Model) panelLogLines() int {
	if m.lastLogLines < 3 {
		return 10
	}
	return m.lastLogLines
}

// jumpMatch scrolls the viewport to the next search match: older (above the
// window) for n, newer for N. No-op without an active search.
func (m *Model) jumpMatch(older bool) {
	if m.search == "" {
		return
	}
	records := m.visibleRecords()
	max := m.panelLogLines()
	start, end := m.window(len(records), max)
	needle := strings.ToLower(m.search)
	if older {
		for i := start - 1; i >= 0; i-- {
			if strings.Contains(strings.ToLower(records[i].Msg), needle) {
				m.scrollOffset = len(records) - i - 1
				m.pinned = true
				return
			}
		}
		return
	}
	for i := end; i < len(records); i++ {
		if strings.Contains(strings.ToLower(records[i].Msg), needle) {
			m.scrollOffset = len(records) - i - 1
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return
		}
	}
}

// recordVisible applies the display level (`d` cycles info/debug/warn), the
// errors-only toggle (`e`), and the output toggle (`o`).
func (m *Model) recordVisible(record recordView) bool {
	if m.levelFilter == "ERRO" { // errors only
		return record.Level == "ERRO" || record.Level == "FATA" || record.Src == int(reporting.SrcStderr)
	}
	if record.Src != int(reporting.SrcLog) {
		// Captured command output: shown at debug display level, or when o
		// forces it on; hidden otherwise.
		return m.displayLevel == "debug" || m.showOutput
	}
	switch m.displayLevel {
	case "debug":
		return true
	case "warn":
		return record.Level == "WARN" || record.Level == "ERRO" || record.Level == "FATA"
	default: // info
		return record.Level != "DEBU"
	}
}

// renderRecord renders one log line from its fields, per the design: time
// (dim) · level glyph · provider (dim) · message · caller (debug only, dim).
// The logger's own formatted text never reaches the screen.
func (m *Model) renderRecord(record recordView) string {
	ascii := m.theme.Glyphs.Done == "+"
	glyph := m.logGlyph(record, ascii)

	var b strings.Builder
	if !record.Time.IsZero() {
		b.WriteString(style(m.theme.Dim).Render(record.Time.Format("15:04:05")) + " ")
	}
	b.WriteString(glyph + " ")
	if record.Provider != "" {
		b.WriteString(style(m.theme.Muted).Render(fmt.Sprintf("%-6s", record.Provider)) + " ")
	}
	msg := normalizeMsg(record.Msg)
	if m.search != "" {
		// Highlight in place rather than filtering the line away.
		if i := strings.Index(strings.ToLower(msg), strings.ToLower(m.search)); i >= 0 {
			msg = msg[:i] + style(m.theme.Accent).Reverse(true).Render(msg[i:i+len(m.search)]) + msg[i+len(m.search):]
		}
	}
	b.WriteString(msg)
	if m.displayLevel == "debug" && record.Caller != "" {
		b.WriteString("  " + style(m.theme.Dim).Render(record.Caller))
	}
	return b.String()
}

// logGlyph maps a record's level and source onto its display glyph.
func (m *Model) logGlyph(record recordView, ascii bool) string {
	switch {
	case record.Src == int(reporting.SrcStderr):
		return style(m.theme.Danger).Render(pick(ascii, ">>", "≫"))
	case record.Src == int(reporting.SrcStdout):
		return style(m.theme.Muted).Render(pick(ascii, ">>", "≫"))
	case record.Level == "ERRO" || record.Level == "FATA":
		return style(m.theme.Danger).Render(pick(ascii, "x", "✗"))
	case record.Level == "WARN":
		return style(m.theme.Warning).Render(pick(ascii, "!", "⚠"))
	case record.Level == "DEBU":
		return style(m.theme.Dim).Render(pick(ascii, ".", "·"))
	default:
		return style(m.theme.Subtext).Render(pick(ascii, ">", "›"))
	}
}

func pick(ascii bool, a, u string) string {
	if ascii {
		return a
	}
	return u
}

// successfullyRe matches rwr's own success boilerplate so the viewport can
// render "installed X" - the provider is already its own column.
var successfullyRe = regexp.MustCompile(`^Successfully (\S+) package (\S+) via \S+$`)

// normalizeMsg compresses known message shapes; unknown messages render
// verbatim. Display-only - the run log keeps the full text.
func normalizeMsg(msg string) string {
	if match := successfullyRe.FindStringSubmatch(msg); match != nil {
		return match[1] + " " + match[2]
	}
	return msg
}

// help bar; collapses to `? help` when space is short.
func (m *Model) viewHelp(short bool) string {
	if short {
		return style(m.theme.Dim).Render(" ? help")
	}
	// Arrows are what the bar advertises; vim keys work but are not required.
	arrows := "↑↓"
	if m.theme.Glyphs.Done == "+" {
		arrows = "arrows"
	}
	keys := " " + arrows + " move · g follow · d level · e errors · o output · / search · +/- size · z zoom · x collapse · m mouse · ? help · q quit"
	if !m.expanded {
		keys = " " + arrows + " move · g follow · d level · e errors · o output · / search · +/- size · z zoom · x expand · m mouse · ? help · q quit"
	}
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
	if m.panelMax {
		// z: the panel takes everything except header and strip; failed rows
		// are the one thing the checklist would never give up, and the strip
		// and header counter still carry them.
		collapsed = nil
	} else if m.manualHeight > 0 {
		// +/-: the user's height wins; pending rows are the first sacrifice
		// (they render at the end of the expanded checklist).
		want := m.manualHeight + 2 // border rows
		excess := 4 + len(collapsed) + want - m.height
		for excess > 0 && len(collapsed) > 0 {
			collapsed = collapsed[:len(collapsed)-1]
			excess--
		}
	}
	for _, line := range collapsed {
		b.WriteString(line + "\n")
	}

	panelHeight := m.height - 4 - len(collapsed)
	if m.manualHeight > 0 && !m.panelMax {
		want := m.manualHeight + 2
		if want < 5 {
			want = 5
		}
		if want < panelHeight {
			panelHeight = want
		}
	}
	if m.panelMax {
		panelHeight = m.height - 3
	}
	b.WriteString(m.viewPanel(panelHeight) + "\n")
	// pending chips - only in collapsed mode; the expanded checklist already
	// has a named row per pending processor.
	if !m.expanded && !m.panelMax {
		var pending []string
		for _, proc := range m.procs {
			if proc.State == ProcPending {
				pending = append(pending, proc.Name)
			}
		}
		if len(pending) > 0 {
			b.WriteString(style(m.theme.Dim).Render(truncate(" pending: "+strings.Join(pending, " "), m.width)) + "\n")
		}
	}
	if m.state == Prompting && m.halt != nil {
		// The halt prompt replaces the help bar; r/R/s/q only exist here.
		reason := ""
		if m.halt.Err != nil {
			reason = truncate(strings.ReplaceAll(m.halt.Err.Error(), "\n", " · "), m.width-40)
		}
		b.WriteString(style(m.theme.Danger).Render(" "+m.theme.Glyphs.Failed+" "+m.halt.Processor+" failed: "+reason) + "\n")
		b.WriteString(style(m.theme.Warning).Render(" r retry · R redo processor · s skip · q abort"))
		return b.String()
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

	// Planned is its own bucket: in dry-run vocabulary it is "would apply",
	// but in a real run a resource still Planned at summary time NEVER RAN
	// (the run died before its processor started). Folding it into "applied"
	// made a run that failed before doing anything report "applied 237".
	applied, skipped, failed, planned, unknown := 0, 0, 0, 0, 0
	for _, res := range m.plan.Resources {
		switch res.Status {
		case types.StatusOK:
			applied++
		case types.StatusPlanned:
			planned++
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
			// c (dry run): changes only - most runs on an existing machine
			// are mostly no-ops, hide the already-present rows.
			if m.changesOnly && res.Status == types.StatusPresent {
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
		b.WriteString(style(m.theme.Muted).Render(fmt.Sprintf(" would apply %d · already present %d · unresolved %d", applied+planned, skipped, unknown)) + "\n")
	} else {
		counts := fmt.Sprintf(" applied %d · skipped %d · failed %d · unknown %d", applied, skipped, failed, unknown)
		if planned > 0 {
			counts += fmt.Sprintf(" · never ran %d", planned)
		}
		b.WriteString(style(m.theme.Muted).Render(counts) + "\n")
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
	// The panel is the selected processor's view, per the design ("the active
	// processor with provider lanes and its log viewport") - the global
	// stream made sequential processors look interleaved. `a` widens to all;
	// explicit scope overrides; otherwise the cursor decides.
	scope := m.scope
	if m.scopeAll {
		scope = ""
	} else if scope == "" && m.cursor < len(m.procs) {
		scope = m.procs[m.cursor].Name
	}
	view := m.views[scope]
	if view == nil {
		view = m.views[""]
	}
	records := m.store.Records(view)
	out := make([]recordView, 0, len(records))
	for _, record := range records {
		out = append(out, recordView{
			Msg: record.Msg, Level: record.Level, Src: int(record.Src),
			Time: record.Time, Provider: record.Provider, Caller: record.Caller,
		})
	}
	return out
}

type recordView struct {
	Msg      string
	Level    string
	Src      int
	Time     time.Time
	Provider string
	Caller   string
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
