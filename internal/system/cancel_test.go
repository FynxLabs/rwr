package system

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fynxlabs/rwr/internal/types"
)

// A run that has been cancelled refuses to start anything new, rather than
// launching every remaining command only to have each one killed.
func TestCancelledRunRefusesToSpawn(t *testing.T) {
	defer BeginRun()()

	Cancel()

	if err := RunCommand(types.Command{Exec: "true"}, false); !errors.Is(err, ErrCancelled) {
		t.Fatalf("RunCommand after cancel = %v, want ErrCancelled", err)
	}
	if _, err := RunCommandOutput(types.Command{Exec: "true"}, false); !errors.Is(err, ErrCancelled) {
		t.Fatalf("RunCommandOutput after cancel = %v, want ErrCancelled", err)
	}
}

// BeginRun's release puts the package back to uncancelled, so a second run in
// the same process is not born dead.
func TestBeginRunResetsCancellation(t *testing.T) {
	release := BeginRun()
	Cancel()
	if !Cancelled() {
		t.Fatal("Cancel did not take effect")
	}
	release()

	defer BeginRun()()
	if Cancelled() {
		t.Fatal("a fresh run started already cancelled")
	}
}

func TestBeginRunResetsOverwriteAll(t *testing.T) {
	overwriteAll.Store(true)
	release := BeginRun()
	if overwriteAll.Load() {
		t.Fatal("BeginRun retained overwrite-all from an earlier run")
	}
	overwriteAll.Store(true)
	release()
	if overwriteAll.Load() {
		t.Fatal("run cleanup retained overwrite-all")
	}
}

// The point of the whole change: a command already running is killed when the
// run is cancelled, and does not have to finish first.
func TestCancelKillsAnInFlightCommand(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("sleep and process groups are posix here")
	}
	defer BeginRun()()

	done := make(chan error, 1)
	go func() {
		// Long enough that finishing on its own would fail the test.
		done <- RunCommand(types.Command{Exec: "sleep", Args: []string{"60"}}, false)
	}()

	// Give it a moment to actually be running before pulling the rug.
	time.Sleep(200 * time.Millisecond)
	started := time.Now()
	Cancel()

	select {
	case err := <-done:
		if !errors.Is(err, ErrCancelled) {
			t.Errorf("killed command reported %v, want ErrCancelled", err)
		}
		if elapsed := time.Since(started); elapsed > 10*time.Second {
			t.Errorf("cancellation took %v; it should not wait for the command", elapsed)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("cancelling did not stop the running command")
	}
}

// Killing the direct child is not enough. A package manager is a shell that
// forks its real work, so cancellation has to take the whole process group -
// otherwise the grandchild carries on holding the terminal it inherited.
func TestCancelKillsGrandchildren(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("process groups are posix")
	}
	defer BeginRun()()

	// The child forks a grandchild that appends to a marker forever. If the
	// group is killed the grandchild dies with it; if only the direct child is
	// killed, the grandchild keeps running and the marker keeps growing.
	dir := t.TempDir()
	marker := filepath.Join(dir, "alive")
	script := "sh -c 'while : ; do echo tick >> " + marker + " ; sleep 0.05 ; done' & wait"

	result := make(chan error, 1)
	go func() {
		result <- RunCommand(types.Command{Exec: "sh", Args: []string{"-c", script}}, false)
	}()

	// Wait for the grandchild to actually be running rather than sleeping a
	// guessed interval. Cancelling before it starts would leave nothing to
	// assert, and the test would pass by skipping the thing it exists for.
	if !waitFor(2*time.Second, func() bool {
		data, err := os.ReadFile(marker) // #nosec G304 -- test-owned temp path
		return err == nil && len(data) > 0
	}) {
		t.Fatal("the grandchild never started, so this test would prove nothing")
	}

	Cancel()

	// The command itself has to come back, or the kill did not reach it.
	select {
	case <-result:
	case <-time.After(20 * time.Second):
		t.Fatal("cancelling did not stop the command")
	}

	// Let anything that survived keep writing, then compare.
	settled, err := os.ReadFile(marker) // #nosec G304 -- test-owned temp path
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	after, err := os.ReadFile(marker) // #nosec G304 -- test-owned temp path
	if err != nil {
		t.Fatal(err)
	}

	if len(after) > len(settled) {
		t.Errorf("the grandchild outlived cancellation: marker grew from %d to %d bytes", len(settled), len(after))
	}
}

// waitFor polls until done returns true, or the deadline passes.
func waitFor(limit time.Duration, done func() bool) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if done() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// A command killed by cancellation exits non-zero like any other failure.
// Reporting that as one would fill a summary with "signal: killed" for work
// the operator deliberately stopped.
func TestKilledCommandReportsCancellationNotFailure(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("sleep is posix here")
	}
	defer BeginRun()()

	done := make(chan error, 1)
	go func() {
		done <- RunCommand(types.Command{Exec: "sleep", Args: []string{"60"}}, false)
	}()
	time.Sleep(200 * time.Millisecond)
	Cancel()

	select {
	case err := <-done:
		if !errors.Is(err, ErrCancelled) {
			t.Fatalf("err = %v, want ErrCancelled", err)
		}
		if strings.Contains(err.Error(), "signal:") || strings.Contains(err.Error(), "killed") {
			t.Errorf("cancellation surfaced as a kill signal: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("cancelling did not stop the running command")
	}
}
