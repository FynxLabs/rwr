package tui

import (
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
	zone "github.com/lrstanley/bubblezone/v2"
)

// State is where the run is.
type State int

const (
	Resolving State = iota
	Running
	Suspended // terminal handed to a child process
	Prompting // interactive error halt
	SummaryState
)

// ProcState is one processor's worst-lane-derived state.
type ProcState int

const (
	ProcPending ProcState = iota
	ProcRunning
	ProcDone
	ProcDegraded
	ProcFailed
	ProcSkipped
)

// Proc is one processor row.
type Proc struct {
	Name    string
	State   ProcState
	Err     error
	Dur     time.Duration
	Started time.Time
	Lanes   map[string]*Lane
}

// Lane is per-provider progress inside a processor. The rendered fill is a
// spring toward the real fraction, so bars glide instead of jumping; ticks
// stop driving it once every spring settles.
type Lane struct {
	Provider string
	Done     int
	Total    int
	Status   types.Status

	// Progress trackers are per blueprint file, so their counts reset when a
	// processor's next file starts; doneBase/lastDone accumulate across files
	// so the bar never jumps backwards mid-processor.
	doneBase int
	lastDone int

	spring   harmonica.Spring
	fill     float64
	velocity float64
}

// settled reports whether the spring has caught up with reality.
func (l *Lane) settled() bool {
	target := l.target()
	delta := l.fill - target
	return delta > -0.005 && delta < 0.005 && l.velocity > -0.01 && l.velocity < 0.01
}

func (l *Lane) target() float64 {
	if l.Total <= 0 {
		return 0
	}
	return float64(l.Done) / float64(l.Total)
}

// state derives a processor's display state from the worst lane state -
// never set directly, per the design.
func (p *Proc) derive() ProcState {
	if p.State == ProcFailed || p.State == ProcSkipped || p.State == ProcPending {
		return p.State
	}
	worst := p.State
	for _, lane := range p.Lanes {
		if lane.Status == types.StatusFailed {
			if p.State == ProcDone {
				return ProcDegraded // some lanes ok, some failed
			}
			worst = ProcDegraded
		}
	}
	return worst
}

// Model is the whole TUI.
type Model struct {
	theme  Theme
	plan   *types.Plan
	procs  []*Proc
	order  map[string]int
	cursor int
	live   int
	pinned bool

	store *reporting.Store
	views map[string]*reporting.View
	scope string // "" = all

	levelFilter  string // "" = all; "ERRO" = errors only (e)
	displayLevel string // "info" (default) | "debug" | "warn"; d cycles
	showOutput   bool   // o: show captured command output outside debug level
	search       string
	searching    bool

	state    State
	errs     []types.StepError
	started  time.Time
	finished time.Time
	dryRun   bool

	width, height int
	compact       bool
	spinnerFrame  int
	focused       bool
	notifyMin     time.Duration
	noNotify      bool

	runLogPath string
	// cancelled records that the operator stopped the run, so the summary
	// can say so rather than presenting a truncated run as a finished one.
	cancelled  bool
	summaryTab int

	// scrollOffset is lines back from the tail (0 = follow); manualHeight is
	// the user's +/- panel size (0 = automatic); panelMax is the z toggle.
	scrollOffset int
	manualHeight int
	panelMax     bool
	scopeAll     bool // a: all-processor log scope instead of the selected one
	changesOnly  bool // c: dry-run summary hides already-present rows
	lastLogLines int  // log capacity of the last rendered panel

	// halt is the pending interactive-halt request while state == Prompting.
	halt *reporting.HaltReq
	// resumedAt is when the terminal last came back from a child process or
	// a prompt entered Prompting; plain keys within the grace window after
	// it are typeahead, not commands.
	resumedAt time.Time
	// expanded shows the full processor checklist (one named row each);
	// collapsed compresses successes onto one line for a bigger log panel.
	// The checklist is the default - a run whose progress list is invisible
	// until you hover an anonymous strip cell reads as frozen.
	expanded bool

	// zones maps rendered strip cells and list rows back to processors, so
	// mouse work survives layout changes instead of hardcoding coordinates.
	zones   *zone.Manager
	hovered int // strip index under the pointer; -1 when none

	// mouseCapture drives the frame's MouseMode; m toggles it off so the
	// terminal's native selection and copy work without holding Shift.
	mouseCapture bool
}

// event wraps a reporting.Event for the tea loop.
type event struct{ e reporting.Event }

// tick drives the spinner and clock.
type tick time.Time

// New builds the model for a run over plan.
func New(theme Theme, plan *types.Plan, store *reporting.Store, dryRun bool, runLogPath string) *Model {
	m := &Model{
		theme:        theme,
		plan:         plan,
		store:        store,
		views:        map[string]*reporting.View{"": store.NewView("")},
		state:        Resolving,
		started:      time.Now(),
		dryRun:       dryRun,
		runLogPath:   runLogPath,
		order:        map[string]int{},
		focused:      true,
		notifyMin:    30 * time.Second,
		zones:        zone.New(),
		hovered:      -1,
		expanded:     true,
		mouseCapture: true,
		displayLevel: "info",
	}
	for i, name := range plan.Order {
		proc := &Proc{Name: name, State: ProcPending, Lanes: map[string]*Lane{}}
		m.procs = append(m.procs, proc)
		m.order[name] = i
		m.views[name] = store.NewView(name)
	}
	// Lanes come from the resolved plan, not just from runtime events: stage 2
	// enumerated every declared resource, so each processor's configured
	// providers render with their real denominators from the first frame
	// ("brew 0/50 · cargo 0/24"), instead of lanes popping in as the executor
	// happens to reach them. LaneUpdate refines Done/Total as the run
	// progresses (profiles can filter the runtime count below the plan's).
	for _, res := range plan.Resources {
		i, ok := m.order[res.Processor]
		if !ok {
			continue
		}
		name := res.Provider
		if name == "" {
			name = "items"
		}
		lane, ok := m.procs[i].Lanes[name]
		if !ok {
			lane = &Lane{Provider: name, spring: harmonica.NewSpring(harmonica.FPS(8), 6.0, 0.9)}
			m.procs[i].Lanes[name] = lane
		}
		lane.Total++
	}
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tick(t) })
}

// Apply consumes a run event (called from the reporter via p.Send).
func (m *Model) apply(e reporting.Event) tea.Cmd {
	switch ev := e.(type) {
	case reporting.ProcStarted:
		m.state = Running
		// Execution is sequential: exactly one processor runs at a time. A
		// second ProcStarted before the prior ProcFinished means the executor
		// stopped emitting Finished - the bug that showed every started
		// processor spinning forever. Fail loudly, never silently.
		for _, proc := range m.procs {
			if proc.State == ProcRunning && proc.Name != ev.Processor {
				log.Errorf("TUI invariant broken: %s started while %s is still marked running (missing ProcFinished)", ev.Processor, proc.Name)
			}
		}
		if i, ok := m.order[ev.Processor]; ok {
			if m.procs[i].State == ProcPending {
				m.procs[i].State = ProcRunning
				m.procs[i].Started = time.Now()
			}
			m.live = i
			if !m.pinned {
				m.cursor = i
			}
		}
	case reporting.ProcFinished:
		if i, ok := m.order[ev.Processor]; ok {
			proc := m.procs[i]
			proc.Dur = ev.Dur
			if ev.Err != nil {
				proc.State = ProcFailed
				proc.Err = ev.Err
			} else {
				proc.State = ProcDone
			}
			proc.State = proc.derive()
		}
	case reporting.ProcSkipped:
		if i, ok := m.order[ev.Processor]; ok {
			m.procs[i].State = ProcSkipped
		}
	case reporting.LaneUpdate:
		if i, ok := m.order[ev.Processor]; ok {
			name := ev.Provider
			if name == "" {
				name = "items"
			}
			lane, ok := m.procs[i].Lanes[name]
			if !ok {
				// Fresh-machine case: stage 2 ran before the package manager
				// was installed, so unpinned entries planned into an "items"
				// lane no runtime update will ever key. Unpinned entries all
				// resolve to the one default provider, so when a provider
				// update arrives with no lane of its own and the "items" lane
				// is untouched, that lane IS those entries - re-key it
				// instead of leaving a ghost pending at 0/N forever.
				if ghost, has := m.procs[i].Lanes["items"]; has && name != "items" &&
					ghost.Done == 0 && ghost.doneBase == 0 {
					delete(m.procs[i].Lanes, "items")
					ghost.Provider = name
					m.procs[i].Lanes[name] = ghost
					lane = ghost
				} else {
					lane = &Lane{Provider: name, spring: harmonica.NewSpring(harmonica.FPS(8), 6.0, 0.9)}
					m.procs[i].Lanes[name] = lane
				}
			}
			// Counts accumulate across a processor's blueprint files: each
			// file gets its own tracker starting at zero, and overwriting
			// with the new file's counts made the bar jump backwards and
			// replaced the plan's cross-file denominator.
			if ev.Done < lane.lastDone {
				lane.doneBase += lane.lastDone
			}
			lane.lastDone = ev.Done
			lane.Done = lane.doneBase + ev.Done
			if total := lane.doneBase + ev.Total; total > lane.Total {
				lane.Total = total
			}
			lane.Status = ev.Status
		}
	case reporting.ResourceDone:
		// Move the matching planned resource to its outcome so Summary
		// renders results, not the plan; unplanned work (imports, entries
		// resolved at run time) is appended.
		res := ev.Resource
		matched := false
		for i := range m.plan.Resources {
			planned := &m.plan.Resources[i]
			if planned.Status != types.StatusPlanned || planned.Processor != res.Processor || planned.Name != res.Name {
				continue
			}
			if planned.Location != "" {
				if res.Location == "" || filepath.Clean(planned.Location) != filepath.Clean(res.Location) {
					continue
				}
			}
			planned.Provider, planned.Status, planned.Detail, planned.Dur = res.Provider, res.Status, res.Detail, res.Dur
			matched = true
			break
		}
		if !matched {
			m.plan.Resources = append(m.plan.Resources, res)
		}
	case reporting.TerminalReq:
		m.state = Suspended
		done := ev.Done
		// The claim is taken inside the exec itself (which bubbletea runs
		// inline on the event loop, where the program cannot die mid-way):
		// claiming any earlier risks the program exiting between claim and
		// exec, leaving a claimed-but-never-serviced request whose waiter
		// blocks forever.
		return tea.Exec(claimedExec{cmd: ev.Cmd, claim: ev.Claim}, func(err error) tea.Msg {
			done <- err
			return execDone{}
		})
	case reporting.HaltReq:
		// Interactive halt: jump to the failing processor, unpin, apply
		// errors-only so the operator sees why, and wait for r/R/s/q. The
		// grace window keeps buffered typeahead from answering the prompt
		// before the operator has even seen it.
		m.state = Prompting
		m.halt = &ev
		m.resumedAt = time.Now()
		if i, ok := m.order[ev.Processor]; ok {
			m.cursor = i
		}
		m.pinned = false
		m.levelFilter = "ERRO"
		return nil
	case reporting.TerminalFunc:
		// Same handover as TerminalReq, for in-process interactions (huh
		// forms, raw prompts): the program releases the terminal, the
		// interaction runs on it, the dashboard resumes after. Claimed at
		// run time for the same reason as TerminalReq above.
		m.state = Suspended
		run := ev.Run
		claim := ev.Claim
		done := ev.Done
		return tea.Exec(funcExec(func() error {
			if !reporting.TryClaim(claim) {
				return nil // the waiter's fallback already ran it
			}
			return run()
		}), func(err error) tea.Msg {
			done <- err
			return execDone{}
		})
	case reporting.RunFinished:
		// Append, not replace: All() emits its collected step errors when it
		// reaches the end, and the runner emits a second RunFinished carrying
		// the run's own error when the run died early - replacing would let
		// whichever arrived last erase the other.
		m.errs = append(m.errs, ev.Errs...)
		m.finished = time.Now()
		m.state = SummaryState
		return m.maybeNotify()
	}
	return nil
}

type execDone struct{}

// claimedExec wraps an *exec.Cmd as a tea.ExecCommand whose Run first takes
// the request's claim: if the waiter's terminal-lost fallback already ran
// the command, Run is a no-op instead of a second execution. The io setters
// mirror bubbletea's own osExecCommand (fill only when unset, so a preset
// Stdin survives).
type claimedExec struct {
	cmd   *exec.Cmd
	claim *atomic.Bool
}

func (c claimedExec) Run() error {
	if !reporting.TryClaim(c.claim) {
		return nil
	}
	return c.cmd.Run()
}

func (c claimedExec) SetStdin(r io.Reader) {
	if c.cmd.Stdin == nil {
		c.cmd.Stdin = r
	}
}

func (c claimedExec) SetStdout(w io.Writer) {
	if c.cmd.Stdout == nil {
		c.cmd.Stdout = w
	}
}

func (c claimedExec) SetStderr(w io.Writer) {
	if c.cmd.Stderr == nil {
		c.cmd.Stderr = w
	}
}

// funcExec adapts a plain func to tea.ExecCommand so tea.Exec can hand the
// terminal to an in-process interaction. The io setters are no-ops: the
// interaction (a huh form, a stdin read) talks to the real terminal, which
// tea has released for the duration of Run.
type funcExec func() error

func (f funcExec) Run() error        { return f() }
func (funcExec) SetStdin(io.Reader)  {}
func (funcExec) SetStdout(io.Writer) {}
func (funcExec) SetStderr(io.Writer) {}

// stripZone and rowZone name a processor's strip cell and list row; the ids
// differ because a Mark of the same id overwrites the earlier zone.
func stripZone(i int) string { return "strip:" + strconv.Itoa(i) }
func rowZone(i int) string   { return "row:" + strconv.Itoa(i) }

// procAt maps a mouse message to the processor whose zone contains it, or -1.
func (m *Model) procAt(msg tea.MouseMsg) int {
	for i := range m.procs {
		for _, id := range []string{stripZone(i), rowZone(i)} {
			if info := m.zones.Get(id); info != nil && !info.IsZero() && info.InBounds(msg) {
				return i
			}
		}
	}
	return -1
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case event:
		return m, m.apply(msg.e)
	case execDone:
		m.state = Running
		// Typeahead guard: keystrokes buffered while a child owned the
		// terminal (extra password characters, a stray Enter) are delivered
		// to the dashboard the moment it resumes. Without a grace period a
		// leftover 'q' quit the run and a leftover 's' answered a halt
		// prompt nobody saw.
		m.resumedAt = time.Now()
		return m, nil
	case tick:
		m.spinnerFrame++
		// Advance the bar springs; when every spring has settled and nothing
		// is running, the animation has nothing left to do - kill the tick
		// (the next event restarts it).
		animating := false
		for _, proc := range m.procs {
			if proc.State == ProcRunning {
				animating = true
			}
			for _, lane := range proc.Lanes {
				lane.fill, lane.velocity = lane.spring.Update(lane.fill, lane.velocity, lane.target())
				if !lane.settled() {
					animating = true
				}
			}
		}
		if !animating && m.state == SummaryState {
			return m, nil
		}
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// Either threshold trips compact mode; resize promotes/demotes live
		// with no state lost - both layouts render from this one model.
		m.compact = msg.Width < 60 || msg.Height < 14
		return m, nil
	case tea.FocusMsg:
		m.focused = true
		return m, nil
	case tea.BlurMsg:
		m.focused = false
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.MouseClickMsg:
		// A click on a strip cell or a processor row selects that processor's
		// block; any click pins. Zones map rendered cells back to processors,
		// so selection survives layout changes.
		m.pinned = true
		if i := m.procAt(msg); i >= 0 {
			m.cursor = i
		}
		return m, nil
	case tea.MouseMotionMsg:
		m.hovered = m.procAt(msg)
		return m, nil
	case tea.MouseWheelMsg:
		// Wheel scrolls the log viewport, 3 lines per tick; scrolling up
		// disengages follow, scrolling back to the bottom re-engages it.
		m.pinned = true
		if msg.Button == tea.MouseWheelUp {
			m.scrollOffset += 3
		} else {
			m.scrollOffset -= 3
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Typeahead grace: within 750ms of resuming from a terminal handover (or
	// entering a halt prompt), plain keys are leftover input from the child -
	// password characters, stray Enters - not commands. ctrl+c always works.
	if msg.String() != "ctrl+c" && !m.resumedAt.IsZero() && time.Since(m.resumedAt) < 750*time.Millisecond {
		return m, nil
	}

	if m.searching {
		switch msg.String() {
		case "enter", "esc":
			m.searching = false
		case "backspace":
			if len(m.search) > 0 {
				m.search = m.search[:len(m.search)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.search += msg.String()
			}
		}
		return m, nil
	}

	// Prompting: r/R retry, s skip, q abort - these exist only at a halt.
	// The claim guarantees exactly one answer: if the terminal-lost
	// fallback already aborted this halt, the keypress must not answer too.
	if m.state == Prompting && m.halt != nil {
		answer := func(d reporting.HaltDecision) {
			if reporting.TryClaim(m.halt.Claim) {
				m.halt.Decision <- d
			}
			m.halt = nil
			m.state = Running
		}
		switch msg.String() {
		case "r", "R":
			answer(reporting.HaltRetry)
			m.levelFilter = ""
			return m, nil
		case "s":
			answer(reporting.HaltSkip)
			m.levelFilter = ""
			return m, nil
		case "q", "ctrl+c":
			// Abort the run; the summary follows via RunFinished.
			answer(reporting.HaltAbort)
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		// Cancel the run, do not merely close the window over it.
		//
		// These used to be tea.Quit alone: the dashboard went away and the run
		// carried on installing as root, with its output landing on a terminal
		// the display had left in raw mode. An operator pressing ctrl-c is
		// asking rwr to stop, and had no way to make that happen.
		//
		// Quit follows, so the summary is not what the operator waits for
		// after asking to leave.
		m.cancelled = true
		system.Cancel()
		return m, tea.Quit
	case "j", "down":
		m.pinned = true
		if m.cursor < len(m.procs)-1 {
			m.cursor++
		}
	case "k", "up":
		m.pinned = true
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "f", "end":
		// Follow: unpin, snap to live, tail the log.
		m.pinned = false
		m.cursor = m.live
		m.scrollOffset = 0
	case "pgup":
		m.pinned = true
		m.scrollOffset += m.panelLogLines()
	case "pgdn":
		m.pinned = true
		m.scrollOffset -= m.panelLogLines()
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
	case "home":
		m.pinned = true
		m.scrollOffset = 1 << 30 // clamped to the top at render
	case "e":
		m.levelFilter = toggle(m.levelFilter, "ERRO")
	case "E":
		// Jump to the first failure with errors-only applied.
		for i, proc := range m.procs {
			if proc.State == ProcFailed || proc.State == ProcDegraded {
				m.cursor, m.pinned = i, true
				break
			}
		}
		m.levelFilter = "ERRO"
	case "o":
		m.showOutput = !m.showOutput
	case "a":
		// Scope: selected processor <-> all processors.
		m.scopeAll = !m.scopeAll
	case "d":
		switch m.displayLevel {
		case "info":
			m.displayLevel = "debug"
		case "debug":
			m.displayLevel = "warn"
		default:
			m.displayLevel = "info"
		}
	case "/":
		m.searching = true
		m.search = ""
	case "esc":
		m.search = ""
	case "n":
		m.jumpMatch(true)
	case "N":
		m.jumpMatch(false)
	case "x":
		m.expanded = !m.expanded
	case "m":
		m.mouseCapture = !m.mouseCapture
		if !m.mouseCapture {
			m.hovered = -1
		}
	case "+", "=":
		if m.manualHeight == 0 {
			m.manualHeight = m.panelLogLines()
		}
		m.manualHeight++
	case "-":
		if m.manualHeight == 0 {
			m.manualHeight = m.panelLogLines()
		}
		if m.manualHeight > 3 {
			m.manualHeight--
		}
	case "z":
		m.panelMax = !m.panelMax
	case "y":
		// Yank the visible viewport lines (plain text) to the clipboard.
		return m, tea.SetClipboard(strings.Join(m.plainLines(m.panelLogLines()), "\n"))
	case "Y":
		if m.runLogPath != "" {
			return m, tea.SetClipboard(m.runLogPath)
		}
	case "c":
		if m.dryRun {
			m.changesOnly = !m.changesOnly
		}
	case "tab", "l", "right":
		if m.state == SummaryState {
			m.summaryTab = (m.summaryTab + 1) % (len(m.procs) + 1)
		}
	case "h", "left":
		if m.state == SummaryState {
			m.summaryTab = (m.summaryTab + len(m.procs)) % (len(m.procs) + 1)
		}
	}
	return m, nil
}

func toggle(current, value string) string {
	if current == value {
		return ""
	}
	return value
}

// maybeNotify emits one OSC 9 notification at completion: only for runs past
// the threshold, only unfocused, only on a recognized terminal, never when
// disabled. Unknown terminals sometimes print unrecognized OSC payloads as
// literal text into the view, so the gate is allow-list shaped.
func (m *Model) maybeNotify() tea.Cmd {
	if m.noNotify || m.focused || time.Since(m.started) < m.notifyMin || !recognizedTerminal() {
		return nil
	}
	failed := len(m.errs)
	body := "rwr finished"
	if failed > 0 {
		body = "rwr finished · " + itoa(failed) + " failed"
	}
	return func() tea.Msg {
		// OSC 9 travels over SSH: provisioning a remote box notifies the
		// terminal you are actually sitting at.
		print("\x1b]9;" + body + "\x07")
		return nil
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
