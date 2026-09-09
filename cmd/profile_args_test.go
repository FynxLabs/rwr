package cmd

import (
	"reflect"
	"testing"

	"github.com/fynxlabs/rwr/internal/helpers"
)

func TestProfileLists(t *testing.T) {
	for _, args := range [][]string{
		{"--profile", "nvidia,desktop"},
		{"--profile", "nvidia, desktop"},
		{"--profile", "nvidia,", "desktop"},
		{"--profile=nvidia,", "desktop"},
		{"-p", "nvidia,", "desktop"},
		{"-pnvidia,", "desktop"},
		{"--profile", "nvidia", "--profile", "desktop"},
		{"--profile", "nvidia", ",", "desktop"},
	} {
		t.Run(args[0]+" "+args[1], func(t *testing.T) {
			app := NewAppConfig()
			root := NewRootCmd(app)
			normalized, err := normalizeProfileArgs(args)
			if err != nil {
				t.Fatal(err)
			}
			if err := root.PersistentFlags().Parse(normalized); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(app.Profiles, []string{"nvidia", "desktop"}) {
				t.Fatalf("profiles=%q, want separate nvidia and desktop", app.Profiles)
			}
			if leftovers := root.PersistentFlags().Args(); len(leftovers) != 0 {
				t.Fatalf("ignored profile arguments: %q", leftovers)
			}
			for _, name := range []string{"nvidia", "desktop"} {
				if !helpers.ShouldInclude([]string{name}, app.Profiles) {
					t.Fatalf("profile %s did not activate", name)
				}
			}
			if helpers.ShouldInclude([]string{"laptop"}, app.Profiles) {
				t.Fatal("unselected profile activated")
			}
		})
	}
}

func TestProfileArgumentsRejectEmptyNames(t *testing.T) {
	for _, args := range [][]string{
		{"--profile"}, {"--profile="}, {"--profile", "nvidia,"},
		{"--profile", "nvidia,", "--debug"}, {"-p", ",desktop"},
		{"--profile", "nvidia,,desktop"},
	} {
		if _, err := normalizeProfileArgs(args); err == nil {
			t.Fatalf("accepted incomplete profile list %q", args)
		}
	}
}

func TestProfileArgumentsPreserveOtherArguments(t *testing.T) {
	args := []string{"run", "scripts", "--init-file", "a,b.cue", "--profile", "nvidia,", "desktop", "--debug", "--", "--profile", "untouched,"}
	want := []string{"run", "scripts", "--init-file", "a,b.cue", "--profile=nvidia,desktop", "--debug", "--", "--profile", "untouched,"}
	got, err := normalizeProfileArgs(args)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}
