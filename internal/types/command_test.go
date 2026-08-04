package types

import (
	"strings"
	"testing"
)

// Some tools take a credential only as an argument, so it cannot always be moved
// to Stdin - but it must still never reach a log file. buildCommand logs argv at
// debug level and the dry-run path logs it at info.
func TestCommandLogArgs_RedactsSecrets(t *testing.T) {
	cmd := Command{
		Exec:    "choco",
		Args:    []string{"source", "add", "--user=alice", "--password=s3cr3t", "--name=internal"},
		Secrets: []string{"s3cr3t"},
	}

	got := strings.Join(cmd.LogArgs(), " ")
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("log args leak the secret: %s", got)
	}
	if !strings.Contains(got, RedactedPlaceholder) {
		t.Errorf("log args do not show a redaction marker: %s", got)
	}
	for _, want := range []string{"source", "add", "--user=alice", "--name=internal"} {
		if !strings.Contains(got, want) {
			t.Errorf("log args dropped non-secret %q: %s", want, got)
		}
	}
}

func TestCommandLogArgs_LeavesArgsAloneWithoutSecrets(t *testing.T) {
	cmd := Command{Exec: "apt", Args: []string{"install", "git"}}

	got := cmd.LogArgs()
	if len(got) != 2 || got[0] != "install" || got[1] != "git" {
		t.Errorf("LogArgs altered a command with no secrets: %v", got)
	}
}

// An empty entry must not turn every argument into the placeholder.
func TestCommandLogArgs_IgnoresEmptySecret(t *testing.T) {
	cmd := Command{Exec: "apt", Args: []string{"install", "git"}, Secrets: []string{""}}

	if got := strings.Join(cmd.LogArgs(), " "); got != "install git" {
		t.Errorf("empty secret corrupted the args: %q", got)
	}
}
