package system

import (
	"errors"
	"os"
	"os/exec"
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

	// The child writes its grandchild's pid and then waits. If the group is
	// killed the grandchild dies with it; if only the direct child is killed,
	// the grandchild keeps running and its marker file keeps growing.
	dir := t.TempDir()
	marker := filepath.Join(dir, "alive")
	script := "sh -c 'while : ; do echo tick >> " + marker + " ; sleep 0.1 ; done' & echo $! ; wait"

	go func() {
		_ = RunCommand(types.Command{Exec: "sh", Args: []string{"-c", script}}, false)
	}()

	time.Sleep(500 * time.Millisecond)
	Cancel()
	time.Sleep(500 * time.Millisecond)

	before, err := os.ReadFile(marker) // #nosec G304 -- test-owned temp path
	if err != nil {
		t.Skipf("grandchild never started, nothing to assert: %v", err)
	}
	time.Sleep(700 * time.Millisecond)
	after, err := os.ReadFile(marker) // #nosec G304 -- test-owned temp path
	if err != nil {
		t.Fatal(err)
	}

	if len(after) > len(before) {
		t.Errorf("the grandchild outlived cancellation: marker grew from %d to %d bytes", len(before), len(after))
	}
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

// Sanity: the helper is wired to real process groups, not silently a no-op.
func TestCommandsGetTheirOwnProcessGroup(t *testing.T) {
	if runtime.GOOS == types.OSWindows {
		t.Skip("process groups are posix")
	}
	built := exec.Command("true")
	intoOwnProcessGroup(built)
	if built.SysProcAttr == nil || !built.SysProcAttr.Setpgid {
		t.Fatal("commands are not placed in their own process group")
	}
}
