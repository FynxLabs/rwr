package cmd

import (
	"strings"
	"testing"
)

// The command is wiring only (the implementation and its tests live in
// internal/convert); what belongs to the command is flag validation.
func TestConvert_RequiresSomethingToDo(t *testing.T) {
	cmd := newConvertCmd()
	cmd.SetArgs([]string{t.TempDir()})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "nothing to do") {
		t.Fatalf("err = %v, want the nothing-to-do error", err)
	}
}

func TestConvert_RejectsUnknownFormat(t *testing.T) {
	cmd := newConvertCmd()
	cmd.SetArgs([]string{"--to", "ini", t.TempDir()})
	if err := cmd.Execute(); err == nil {
		t.Fatal("unknown format accepted")
	}
}
