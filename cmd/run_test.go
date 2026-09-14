package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

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

func TestRunHandlesBlueprintSyncInsideDashboard(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "rwr"}
	run := &cobra.Command{Use: "run"}
	root.AddCommand(run)
	all := &cobra.Command{Use: "all"}
	packages := &cobra.Command{Use: "packages"}
	bootstrap := &cobra.Command{Use: "bootstrap"}
	run.AddCommand(all, packages, bootstrap)

	for _, tc := range []struct {
		name string
		cmd  *cobra.Command
		args []string
		want bool
	}{
		{name: "run all", cmd: all, want: true},
		{name: "run packages", cmd: packages, want: true},
		{name: "bootstrap needs pre-run sync", cmd: bootstrap, want: false},
		{name: "root shorthand", cmd: root, args: []string{"files"}, want: true},
		{name: "root bootstrap", cmd: root, args: []string{"bootstrap"}, want: false},
		{name: "root without shorthand", cmd: root, want: false},
		{name: "unrelated command", cmd: &cobra.Command{Use: "status"}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := runHandlesBlueprintSync(tc.cmd, tc.args); got != tc.want {
				t.Fatalf("runHandlesBlueprintSync() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCredentialsInProcessorTable(t *testing.T) {
	p, ok := processorShorthand("credentials")
	if !ok || p.blueprint != "credentials" || p.bootstrap || p.githubAuth {
		t.Fatalf("invalid credentials dispatch %+v", p)
	}
	run := newRunCmd(&AppConfig{})
	sub, _, err := run.Find([]string{"credentials"})
	if err != nil || sub.Name() != "credentials" {
		t.Fatal("credentials command missing")
	}
}

func TestOmarchyUsesConfigurationProcessor(t *testing.T) {
	if _, ok := processorShorthand("omarchy"); ok {
		t.Fatal("omarchy must be a configuration tool, not a standalone processor")
	}
	p, ok := processorShorthand("configuration")
	if !ok || p.blueprint != "configuration" {
		t.Fatalf("configuration processor missing: %+v", p)
	}
}

func TestExceptFlagRepeatedAndCommaSeparated(t *testing.T) {
	app := &AppConfig{}
	root := &cobra.Command{Use: "rwr"}
	registerRootFlags(root, app)
	if err := root.ParseFlags([]string{"--except", "credentials,fonts", "--except", "repository"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"credentials", "fonts", "repository"}
	if len(app.Except) != len(want) {
		t.Fatalf("except=%v", app.Except)
	}
	for i, name := range want {
		if app.Except[i] != name {
			t.Fatalf("except=%v", app.Except)
		}
	}
}
