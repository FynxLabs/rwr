package omarchy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func ptr[T any](v T) *T { return &v }
func fixture(t *testing.T) (*Client, *[]types.Command) {
	t.Helper()
	home := t.TempDir()
	dist := t.TempDir()
	commands := []types.Command{}
	write := func(path string, raw []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := []byte(`{"version":1,"idle":{"screensaver":150,"lock":300},"bar":{"position":"top","layout":{"left":[{"id":"omarchy.menu"},{"id":"omarchy.workspaces"}],"center":[],"right":[]}},"plugins":[{"id":"example.overlay","preserved":"value"}],"unknown":{"keep":true}}`)
	write(filepath.Join(home, ".config/omarchy/shell.json"), config)
	write(filepath.Join(dist, "config/omarchy/shell.json"), config)
	c := &Client{Home: home, Distribution: dist, LookPath: func(name string) (string, error) { return name, nil }}
	catalog := []Manifest{{ID: "omarchy.menu", Kinds: []string{"bar-widget", "menu"}, FirstParty: true, SourceDir: dist}, {ID: "omarchy.workspaces", Kinds: []string{"bar-widget"}, FirstParty: true, SourceDir: dist}, {ID: "example.overlay", Kinds: []string{"overlay", "bar-widget"}, SourceDir: filepath.Join(home, ".config/omarchy/plugins/example.overlay")}}
	c.Read = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "omarchy" && len(args) >= 2 && args[0] == "plugin" {
			switch args[1] {
			case "catalog":
				return json.Marshal(catalog)
			case "list":
				cfg, _, err := c.config()
				if err != nil {
					return nil, err
				}
				list := append([]Manifest(nil), catalog...)
				for i, p := range list {
					entry, err := pluginEntry(cfg, p.ID, false)
					if err != nil {
						return nil, err
					}
					list[i].Enabled = p.FirstParty || entry != nil
					for _, id := range cfg["disabledPlugins"].([]any) {
						if id == p.ID {
							list[i].Enabled = false
						}
					}
				}
				return json.Marshal(list)
			case "validate":
				m, err := manifestAt(args[2])
				if err != nil || m["id"] == nil {
					return nil, fmt.Errorf("invalid manifest")
				}
				return []byte("ok"), nil
			}
		}
		return nil, fmt.Errorf("unexpected probe %s %v", name, args)
	}
	// Make disabledPlugins an explicit array for the fixture's runtime model.
	var doc map[string]any
	_ = json.Unmarshal(config, &doc)
	doc["disabledPlugins"] = []any{}
	config, _ = json.Marshal(doc)
	write(filepath.Join(home, ".config/omarchy/shell.json"), config)
	c.Run = func(cmd types.Command) error {
		commands = append(commands, cmd)
		if cmd.Exec == "omarchy" && len(cmd.Args) >= 3 && cmd.Args[0] == "plugin" && (cmd.Args[1] == "enable" || cmd.Args[1] == "disable") {
			cfg, old, err := c.config()
			if err != nil {
				return err
			}
			id := cmd.Args[2]
			disabled := []any{}
			for _, v := range cfg["disabledPlugins"].([]any) {
				if v != id {
					disabled = append(disabled, v)
				}
			}
			if cmd.Args[1] == "disable" {
				disabled = append(disabled, id)
			} else {
				if _, err := pluginEntry(cfg, id, true); err != nil {
					return err
				}
			}
			cfg["disabledPlugins"] = disabled
			raw, _ := json.Marshal(cfg)
			_, err = c.replace(".config/omarchy/shell.json", old, raw, 0600)
			return err
		}
		return nil
	}
	return c, &commands
}
func TestSettingsAndOptionalWidgetConverge(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	cfg, old, err := c.config()
	if err != nil {
		t.Fatal(err)
	}
	bar := cfg["bar"].(map[string]any)
	layout := bar["layout"].(map[string]any)
	layout["right"] = []any{map[string]any{"id": "example.overlay", "preserved": "value"}}
	cfg["plugins"] = []any{}
	raw, _ := json.Marshal(cfg)
	if _, err := c.replace(".config/omarchy/shell.json", old, raw, 0600); err != nil {
		t.Fatal(err)
	}
	ops, err := entryOperations(types.OmarchySetup{Name: "desktop", Plugins: []types.OmarchyPlugin{{ID: "example.overlay", Settings: map[string]any{"hotCornerEnabled": false, "count": 0, "nullable": nil}, Widget: &types.OmarchyWidget{Visible: ptr(false)}}}}, "/blueprint/setup.yaml")
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	apply := func() {
		t.Helper()
		if err := c.Apply(context.Background(), ops, func(r Result) {
			if r.Changed {
				changed++
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	first := changed
	apply()
	if changed != first || first == 0 {
		t.Fatal("repeat apply rewrote config")
	}
	if len(*commands) != 0 {
		t.Fatal("settings restarted shell")
	}
	actual, _, err := c.config()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := pluginEntry(actual, "example.overlay", false)
	if err != nil {
		t.Fatal(err)
	}
	if entry["preserved"] != "value" || entry["hotCornerEnabled"] != false || actual["unknown"] == nil {
		t.Fatal("undeclared options lost")
	}
	if len(actual["bar"].(map[string]any)["layout"].(map[string]any)["left"].([]any)) != 2 {
		t.Fatal("stock workspace widget changed")
	}
}
func TestStrictSchemaAndConflicts(t *testing.T) {
	t.Parallel()
	cases := []string{
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","typo":true}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"install","shell":{"idle":{"lock":300}}}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","schema":"org.example","shell":{"idle":{"lock":300}}}]}`,
		`{"configurations":[{"name":"a","tool":"gsettings","import":"shared.json"}]}`,
		`{"configurations":[{"name":"a","tool":"gsettings","plugins":[{"id":"example.x"}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"../escape"}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"omarchy.menu","state":"absent"}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"example.x","source":{"git":"ext::evil"}}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"example.x","enabled":false,"widget":{"visible":true}}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"example.x","settings":{"id":"other"}}]}]}`,
		`{"configurations":[{"name":"a","tool":"omarchy","action":"set","theme":{"name":"one","active":true}},{"name":"b","tool":"omarchy","action":"set","theme":{"name":"two","active":true}}]}`,
	}
	for _, raw := range cases {
		if _, err := Load([]byte(raw), "json", "/blueprint.json", &types.InitConfig{}); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestOmarchyRejectsExplicitForeignZeroValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, format, raw string
	}{
		{"json false", "json", `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","elevated":false}]}`},
		{"json empty", "json", `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","file":""}]}`},
		{"json null", "json", `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","value":null}]}`},
		{"yaml false", "yaml", "configurations:\n  - name: desktop\n    tool: omarchy\n    action: set\n    run_once: false\n"},
		{"yaml null", "yaml", "configurations:\n  - name: desktop\n    tool: omarchy\n    action: set\n    value: null\n"},
		{"toml empty", "toml", "[[configurations]]\nname = \"desktop\"\ntool = \"omarchy\"\naction = \"set\"\nschema = \"\"\n"},
		{"cue false", "cue", `configurations: [{name: "desktop", tool: "omarchy", action: "set", elevated: false}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(tt.raw), tt.format, "/blueprint."+tt.format, &types.InitConfig{}); err == nil {
				t.Fatalf("accepted explicitly supplied foreign field: %s", tt.raw)
			}
		})
	}
}

func TestImportsAndProfilesUseDeclaringDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "common"), 0700); err != nil {
		t.Fatal(err)
	}
	raw := `{"configurations":[{"name":"shared","tool":"omarchy","action":"set","profiles":["desktop"],"theme":{"name":"mine","source":{"path":"theme"},"active":false}}]}`
	if err := os.WriteFile(filepath.Join(root, "common/shared.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	config := &types.InitConfig{}
	config.Variables.Flags.Profiles = []string{"desktop"}
	ops, err := Load([]byte(`configurations: [{tool: omarchy, import: "common/shared.json"}]`), "yaml", filepath.Join(root, "setup.yaml"), config)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 || ops[0].Theme.Source.Path != filepath.Join(root, "common/theme") {
		t.Fatalf("bad imported path: %+v", ops)
	}
	config.Variables.Flags.Profiles = []string{"other"}
	ops, err = Load([]byte(raw), "json", filepath.Join(root, "common/shared.json"), config)
	if err != nil || len(ops) != 0 {
		t.Fatal("filtered profile planned")
	}
}
func TestInvalidDiscoveryAndConcurrentWritesPreserveConfig(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	before, err := c.read(".config/omarchy/shell.json")
	if err != nil {
		t.Fatal(err)
	}
	c.Read = func(context.Context, string, ...string) ([]byte, error) { return []byte("null"), nil }
	if err := c.Apply(context.Background(), []Operation{{ID: "x", Kind: "default", Path: []string{"editor"}, Value: "nvim"}}, func(Result) {}); err == nil {
		t.Fatal("invalid catalog accepted")
	}
	if len(*commands) > 0 {
		t.Fatal("mutated before discovery")
	}
	if _, err := c.replace(".config/omarchy/shell.json", []byte("stale"), []byte("{}"), 0600); err == nil {
		t.Fatal("concurrent write accepted")
	}
	after, err := c.read(".config/omarchy/shell.json")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("lost existing data")
	}
}
func TestDryRunDoesNotProbeOrWrite(t *testing.T) {
	c, commands := fixture(t)
	c.Read = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("dry-run probed runtime")
		return nil, nil
	}
	system.SetDryRun(true)
	defer system.SetDryRun(false)
	count := 0
	if err := c.Apply(context.Background(), []Operation{{ID: "integration/screensaver", Kind: "screensaver", Screensaver: &types.OmarchyScreensaver{Mode: "stock"}}}, func(Result) { count++ }); err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(*commands) != 0 {
		t.Fatal("dry-run mutated")
	}
}
func TestReadinessFailureDoesNotChangeRoute(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	path := filepath.Join(c.Distribution, "shell/plugins/services/idle/Service.qml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("omarchy-launch-screensaver org.omarchy.screensaver screensaver-dismissed screensaverWindowCount"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(c.Distribution, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver"), []byte("#!/bin/bash"), 0700); err != nil {
		t.Fatal(err)
	}
	c.Run = func(types.Command) error { return errors.New("not ready") }
	_, err := c.screensaver(context.Background(), &types.OmarchyScreensaver{Mode: "external", Launch: &types.OmarchyExec{Exec: "/engine"}, Check: &types.OmarchyExec{Exec: "/engine"}})
	if err == nil {
		t.Fatal("readiness failure accepted")
	}
	if _, err := c.read(routePath); !os.IsNotExist(err) {
		t.Fatal("route was installed before readiness")
	}
}
func TestScreensaverDispatcherPreservesArguments(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	c, _ := fixture(t)
	engine := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(engine, []byte("#!/bin/bash\nprintf '%s\\0' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	args := []string{"start", "space value", "quote'\"", "$(touch should-never-exist)", "line\nbreak"}
	raw := c.routeContent(&types.OmarchyScreensaver{Launch: &types.OmarchyExec{Exec: engine, Args: args}})
	script := filepath.Join(t.TempDir(), "route")
	if err := os.WriteFile(script, raw, 0700); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("bash", script, "force").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join(append(args, "force"), "\x00") + "\x00"
	if string(out) != want {
		t.Fatalf("argv changed: %q", out)
	}
}

func TestValidSchemaBoundaries(t *testing.T) {
	t.Parallel()
	for name, raw := range map[string]string{
		"single active theme":       `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","theme":{"name":"one","active":true}}]}`,
		"disabled hidden widget":    `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","plugins":[{"id":"example.widget","enabled":false,"widget":{"visible":false}}]}]}`,
		"mixed configuration tools": `{"configurations":[{"name":"gnome","tool":"gsettings","action":"set","schema":"org.gnome.desktop.interface","settings":{"color-scheme":"prefer-dark"}},{"name":"desktop","tool":"omarchy","action":"set","shell":{"idle":{"lock":300}}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ops, err := Load([]byte(raw), "json", "/blueprint.json", &types.InitConfig{})
			if err != nil || len(ops) == 0 {
				t.Fatalf("valid blueprint rejected: %v", err)
			}
		})
	}
}
