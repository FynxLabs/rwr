package omarchy

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestExamplesHaveEquivalentPlans(t *testing.T) {
	var reference []Operation
	for _, format := range []string{"json", "yaml", "toml", "cue"} {
		path := filepath.Join("../../examples/omarchy", format, "desktop."+format)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		ops, err := Load(raw, format, path, &types.InitConfig{})
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		for i := range ops {
			ops[i].Origin = ""
		}
		if reference == nil {
			reference = ops
		} else if !equal(reference, ops) {
			t.Fatalf("%s lost optional values or changed plan", format)
		}
	}
}
func TestWorkspaceCycleUsesPluginLimit(t *testing.T) {
	for _, tool := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " unavailable")
		}
	}
	script, err := filepath.Abs("../../examples/omarchy/assets/workspace-cycle")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name           string
		current, limit int
		direction      string
		want           string
		missing        bool
		invalid        bool
	}{
		{"wrap next", 5, 5, "next", "1", false, false}, {"wrap previous", 1, 5, "previous", "5", false, false}, {"updated ring", 7, 7, "next", "1", false, false}, {"next inside", 5, 7, "next", "6", false, false}, {"special workspace", -99, 7, "previous", "7", false, false}, {"missing setting", 10, 0, "next", "1", true, false}, {"invalid zero", 2, 0, "next", "", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home, bin := t.TempDir(), t.TempDir()
			writeFixture(t, filepath.Join(bin, "hyprctl"), []byte("#!/bin/bash\nif [[ $1 == activeworkspace ]]; then printf '%s' "+shellQuote(`{"id":`+stringInt(tt.current)+`}`)+"; else printf '%s' \"$2\"; fi\n"), 0700)
			config := map[string]any{"plugins": []any{map[string]any{"id": "io.github.woogy7.workspaces", "maxWorkspaces": tt.limit}}}
			if tt.missing {
				config = map[string]any{"plugins": []any{}}
			}
			raw, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, filepath.Join(home, ".config/omarchy/shell.json"), raw, 0600)
			cmd := exec.Command("bash", script, tt.direction)
			cmd.Env = append(os.Environ(), "HOME="+home, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, err := cmd.Output()
			if tt.invalid {
				if err == nil {
					t.Fatal("invalid limit dispatched")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), `workspace = "`+tt.want+`"`) {
				t.Fatalf("wrong target: %s", out)
			}
		})
	}
}
func stringInt(n int) string {
	b, err := json.Marshal(n)
	if err != nil {
		return ""
	}
	return string(b)
}
