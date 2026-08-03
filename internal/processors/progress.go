package processors

import (
	"time"

	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/types"
)

// progress tracks per-provider lane counts for one processor and emits the
// events the TUI's lanes fill from: one ResourceDone per completed unit of
// work plus a LaneUpdate carrying the lane's running done/total. Headless
// runs are unaffected — LogReporter ignores both events.
//
// The lane provider key is "" for processors without providers (files,
// services, git, scripts, …); the display layer names that lane.
type progress struct {
	processor string
	done      map[string]int
	total     map[string]int
	failed    map[string]bool
}

func newProgress(processor string) *progress {
	return &progress{
		processor: processor,
		done:      map[string]int{},
		total:     map[string]int{},
		failed:    map[string]bool{},
	}
}

// expect raises a lane's denominator before its items run, so the lane
// renders 0/N instead of growing its total as items complete.
func (p *progress) expect(provider string, n int) {
	if n <= 0 {
		return
	}
	p.total[provider] += n
	p.emitLane(provider)
}

// item reports one completed unit of work. A done count past the expected
// total (imports, entries resolved at run time) grows the total: the lane
// never shows 7/5.
func (p *progress) item(provider, name, action string, status types.Status, detail string, dur time.Duration) {
	p.itemIdentity(provider, name, action, status, detail, dur, nil)
}

// itemIdentity is item with extra journal identity — the fields a later
// uninstall needs to find the thing again (a file's dest and sha256, a git
// checkout's target). provider+name always land in the identity.
func (p *progress) itemIdentity(provider, name, action string, status types.Status, detail string, dur time.Duration, identity map[string]string) {
	p.done[provider]++
	if p.done[provider] > p.total[provider] {
		p.total[provider] = p.done[provider]
	}
	if status == types.StatusFailed {
		p.failed[provider] = true
	}
	reporting.Emit(reporting.ResourceDone{Resource: types.Resource{
		Processor: p.processor,
		Provider:  provider,
		Name:      name,
		Action:    action,
		Status:    status,
		Detail:    detail,
		Dur:       dur,
	}})
	p.emitLane(provider)
	journalAppend(p.processor, provider, name, action, status, detail, identity)
}

func (p *progress) emitLane(provider string) {
	status := types.StatusOK
	if p.failed[provider] {
		status = types.StatusFailed
	}
	reporting.Emit(reporting.LaneUpdate{
		Processor: p.processor,
		Provider:  provider,
		Done:      p.done[provider],
		Total:     p.total[provider],
		Status:    status,
	})
}
