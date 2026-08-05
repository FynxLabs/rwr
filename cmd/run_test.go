package cmd

import "testing"

// bootstrap resolves both as `rwr run bootstrap` and as the root shorthand
// `rwr bootstrap`, like every other processor in the table.
func TestBootstrapInProcessorTable(t *testing.T) {
	spec, ok := processorShorthand("bootstrap")
	if !ok {
		t.Fatal("root shorthand `rwr bootstrap` does not resolve")
	}
	if !spec.bootstrap {
		t.Fatal("bootstrap spec does not use the dedicated dispatch")
	}

	runCmd := newRunCmd(&AppConfig{})
	for _, sub := range runCmd.Commands() {
		if sub.Name() == "bootstrap" {
			return
		}
	}
	t.Fatal("`rwr run bootstrap` subcommand missing")
}

// hasProcessorArg gates the "skip init for bare `rwr`" check in
// PersistentPreRunE. A bare invocation must not initialize a tree; any
// recognized processor shorthand - or "all" - must.
func TestHasProcessorArg(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"packages"}, true},
		{[]string{"files"}, true},
		{[]string{"bootstrap"}, true},
		{[]string{"all"}, true},
		{[]string{"nonexistent"}, false},
		{[]string{"packages", "extra"}, true}, // first arg decides
	}
	for _, tc := range cases {
		if got := hasProcessorArg(tc.args); got != tc.want {
			t.Errorf("hasProcessorArg(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
