package processors

import (
	"errors"
	"strings"
	"testing"
)

func TestFailureLedger_CleanRunReportsNoError(t *testing.T) {
	resetFailures()

	if got := failureCount(); got != 0 {
		t.Fatalf("fresh ledger has %d failures, want 0", got)
	}
	if err := failureError(); err != nil {
		t.Errorf("clean run returned %v, want nil", err)
	}
}

// The failure this guards against: every package fails to install, each one is
// logged and skipped, and the run still exits 0.
func TestFailureLedger_RecordedFailuresSurface(t *testing.T) {
	resetFailures()
	t.Cleanup(resetFailures)

	recordFailure("packages", "git", errors.New("not found in repositories"))
	recordFailure("ssh_keys", "id_ed25519", errors.New("permission denied"))

	if got := failureCount(); got != 2 {
		t.Fatalf("ledger recorded %d failures, want 2", got)
	}

	err := failureError()
	if err == nil {
		t.Fatal("recorded failures produced a nil error")
	}

	for _, want := range []string{"2 operation(s) failed", "packages", "git", "not found in repositories", "ssh_keys", "id_ed25519", "permission denied"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("summary is missing %q:\n%s", want, err)
		}
	}
}

func TestFailureLedger_ResetClearsPreviousRun(t *testing.T) {
	resetFailures()
	recordFailure("packages", "vim", errors.New("boom"))
	resetFailures()

	if err := failureError(); err != nil {
		t.Errorf("reset ledger still reports %v", err)
	}
}
