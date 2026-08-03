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
