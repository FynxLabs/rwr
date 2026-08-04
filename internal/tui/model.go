package tui

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/fynxlabs/rwr/internal/reporting"
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

	levelFilter  string // "" = all; "ERRO" = errors only
	stdoutFilter bool
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
	summaryTab int
	expanded   bool // collapsed-success line expanded

	// zones maps rendered strip cells and list rows back to processors, so
	// mouse work survives layout changes instead of hardcoding coordinates.
	zones   *zone.Manager
	hovered int // strip index under the pointer; -1 when none
}

// event wraps a reporting.Event for the tea loop.
type event struct{ e reporting.Event }

// tick drives the spinner and clock.
type tick time.Time

// New builds the model for a run over plan.
func New(theme Theme, plan *types.Plan, store *reporting.Store, dryRun bool, runLogPath string) *Model {
	m := &Model{
		theme:      theme,
		plan:       plan,
		store:      store,
		views:      map[string]*reporting.View{"": store.NewView("")},
		state:      Resolving,
		started:    time.Now(),
		dryRun:     dryRun,
		runLogPath: runLogPath,
		order:      map[string]int{},
		focused:    true,
		notifyMin:  30 * time.Second,
		zones:      zone.New(),
		hovered:    -1,
	}
	for i, name := range plan.Order {
		proc := &Proc{Name: name, State: ProcPending, Lanes: map[string]*Lane{}}
		m.procs = append(m.procs, proc)
		m.order[name] = i
		m.views[name] = store.NewView(name)
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
				lane = &Lane{Provider: name, spring: harmonica.NewSpring(harmonica.FPS(8), 6.0, 0.9)}
				m.procs[i].Lanes[name] = lane
			}
			lane.Done, lane.Total, lane.Status = ev.Done, ev.Total, ev.Status
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
			planned.Provider, planned.Status, planned.Detail, planned.Dur = res.Provider, res.Status, res.Detail, res.Dur
			matched = true
			break
		}
		if !matched {
			m.plan.Resources = append(m.plan.Resources, res)
		}
	case reporting.TerminalReq:
		m.state = Suspended
		cmd := ev.Cmd
		done := ev.Done
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			done <- err
			return execDone{}
		})
	case reporting.RunFinished:
		m.errs = ev.Errs
		m.finished = time.Now()
		m.state = SummaryState
		return m.maybeNotify()
	}
	return nil
}

type execDone struct{}

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
		m.pinned = true
		return m, nil
	}
	return m, nil
}

func (m *Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	switch msg.String() {
	case "q", "ctrl+c":
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
	case "g":
		// Unpin and snap back to live.
		m.pinned = false
		m.cursor = m.live
	case "e":
		m.levelFilter = toggle(m.levelFilter, "ERRO")
	case "o":
		m.stdoutFilter = !m.stdoutFilter
	case "/":
		m.searching = true
		m.search = ""
	case "x":
		m.expanded = !m.expanded
	case "tab":
		if m.state == SummaryState {
			m.summaryTab = (m.summaryTab + 1) % (len(m.procs) + 1)
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
