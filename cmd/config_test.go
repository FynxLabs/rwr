package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// The config command grew view/edit/create subcommands; --create survives as
// a deprecated alias pointing at the subcommand.
func TestConfigSubcommands(t *testing.T) {
	configCmd := newConfigCmd()
	want := map[string]bool{"view": false, "edit": false, "create": false}
	for _, sub := range configCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("rwr config %s missing", name)
		}
	}

	create := configCmd.Flags().Lookup("create")
	if create == nil || create.Deprecated == "" {
		t.Fatal("--create is not marked deprecated")
	}
	if !strings.Contains(create.Deprecated, "rwr config create") {
		t.Fatalf("deprecation message %q does not name the subcommand", create.Deprecated)
	}
}

// -n and -l are the documented shorts for --dry-run and --log-level.
func TestRootShorthands(t *testing.T) {
	app := &AppConfig{}
	root := &cobra.Command{Use: "rwr"}
	registerRootFlags(root, app)

	if f := root.PersistentFlags().ShorthandLookup("n"); f == nil || f.Name != "dry-run" {
		t.Fatalf("-n is not --dry-run (got %v)", f)
	}
	if f := root.PersistentFlags().ShorthandLookup("l"); f == nil || f.Name != "log-level" {
		t.Fatalf("-l is not --log-level (got %v)", f)
	}
}

// No two flags may claim the same shorthand across the root set and any
// subcommand's local set — a future flag grabbing a taken letter fails here.
func TestNoShorthandCollisions(t *testing.T) {
	app := &AppConfig{}
	root := NewRootCmd(app)

	seen := map[string]string{}
	root.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Shorthand == "" {
			return
		}
		if prev, taken := seen[f.Shorthand]; taken {
			t.Errorf("shorthand -%s claimed by both --%s and --%s", f.Shorthand, prev, f.Name)
		}
		seen[f.Shorthand] = f.Name
	})
	for _, sub := range root.Commands() {
		local := map[string]string{}
		for k, v := range seen {
			local[k] = v
		}
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Shorthand == "" {
				return
			}
			if prev, taken := local[f.Shorthand]; taken {
				t.Errorf("%s: shorthand -%s claimed by both --%s and --%s", sub.Name(), f.Shorthand, prev, f.Name)
			}
			local[f.Shorthand] = f.Name
		})
	}
}
