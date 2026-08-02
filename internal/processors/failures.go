package processors

import (
	"fmt"
	"strings"
	"sync"

	"charm.land/log/v2"
)

// Several processors deliberately keep going when one item fails: a package that
// is not in the repositories should not stop the twenty after it, and a git repo
// that is temporarily unreachable should not abandon the rest of the run. That is
// the right behavior. What was wrong is that those failures then vanished — they
// were logged and the run exited 0 with "RWR Run Complete!", so a run in which
// every single package failed to install was indistinguishable from a clean one.
//
// The ledger records them instead. Processors keep going and keep logging; All()
// asks at the end whether anything failed and returns a non-nil error if so, which
// is what puts it in the exit code.
type runFailures struct {
	mu    sync.Mutex
	items []runFailure
}

type runFailure struct {
	component string
	subject   string
	err       error
}

var failures runFailures

// recordFailure notes a non-fatal failure and logs it. Call it instead of a bare
// log.Warnf/log.Errorf wherever a processor swallows an error and continues.
func recordFailure(component, subject string, err error) {
	failures.mu.Lock()
	failures.items = append(failures.items, runFailure{component: component, subject: subject, err: err})
	failures.mu.Unlock()

	log.Errorf("%s: %s: %v", component, subject, err)
}

// resetFailures clears the ledger at the start of a run.
func resetFailures() {
	failures.mu.Lock()
	failures.items = nil
	failures.mu.Unlock()
}

// failureCount reports how many non-fatal failures the run has accumulated.
func failureCount() int {
	failures.mu.Lock()
	defer failures.mu.Unlock()
	return len(failures.items)
}

// failureError summarizes the ledger as a single error, or nil when the run was
// clean. The summary lists every failure so the exit code is not the only signal.
func failureError() error {
	failures.mu.Lock()
	defer failures.mu.Unlock()

	if len(failures.items) == 0 {
		return nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d operation(s) failed:", len(failures.items))
	for _, item := range failures.items {
		fmt.Fprintf(&b, "\n  - %s: %s: %v", item.component, item.subject, item.err)
	}
	return fmt.Errorf("%s", b.String())
}
