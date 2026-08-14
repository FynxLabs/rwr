package system

import (
	"context"
	"errors"
	"sync"
)

// ErrCancelled is returned by the executor once the run has been cancelled.
// It is distinct so callers can tell "the operator stopped this" from "this
// step failed", which read very differently in a summary.
var ErrCancelled = errors.New("run cancelled")

// Cancellation is package state for the same reason dry-run and the executor
// are: it is a property of the run, not of any one command, and threading a
// context through ten processor signatures to reach exec.Cmd would touch every
// call site to say the same thing at each of them.
//
// Before this existed there was no cancellation at all - no signal handler
// anywhere in the binary, and the dashboard's q/ctrl+c only quit the display
// while the run carried on installing as root. An operator who wanted to stop
// a run had no way to.
var (
	cancelMu  sync.Mutex
	runCtx    = context.Background()
	runCancel context.CancelFunc
)

// BeginRun installs a fresh cancellable context for a run and returns the
// function that releases it. Calling it again replaces the previous one, so a
// second run in the same process starts uncancelled.
func BeginRun() (release func()) {
	cancelMu.Lock()
	defer cancelMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	runCtx, runCancel = ctx, cancel

	return func() {
		cancelMu.Lock()
		defer cancelMu.Unlock()
		cancel()
		runCtx, runCancel = context.Background(), nil
	}
}

// Cancel stops the run: every command still to start refuses, and the one
// currently running is killed. Safe to call repeatedly and from any goroutine,
// which matters because it is reached from a signal handler and from the
// dashboard's key handling at the same time.
func Cancel() {
	cancelMu.Lock()
	cancel := runCancel
	cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Cancelled reports whether the run has been cancelled. Loops that iterate
// items call it so a cancelled run stops promptly instead of grinding through
// hundreds of refusals, one recorded failure at a time.
func Cancelled() bool {
	return RunContext().Err() != nil
}

// RunContext is the run's context, for the command spawn and for anything else
// that wants to stop early.
func RunContext() context.Context {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	return runCtx
}
