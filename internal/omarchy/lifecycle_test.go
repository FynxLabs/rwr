package omarchy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/freehold-digital/rwr/internal/types"
)

func writeFixture(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
}
func dynamicPlugins(t *testing.T, c *Client) {
	t.Helper()
	previous := c.Read
	c.Read = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		raw, err := previous(ctx, name, args...)
		if name != "omarchy" || len(args) < 2 || args[0] != "plugin" || (args[1] != "catalog" && args[1] != "list") {
			return raw, err
		}
		if err != nil {
			return nil, err
		}
		var list []Manifest
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, err
		}
		known := map[string]bool{}
		for _, m := range list {
			known[m.ID] = true
		}
		dirs, err := os.ReadDir(filepath.Join(c.Home, ".config/omarchy/plugins"))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		config, _, err := c.config()
		if err != nil {
			return nil, err
		}
		for _, d := range dirs {
			if !d.IsDir() || strings.HasPrefix(d.Name(), ".") || known[d.Name()] {
				continue
			}
			path := filepath.Join(c.Home, ".config/omarchy/plugins", d.Name())
			raw, err := readExternal(filepath.Join(path, "manifest.json"))
			if err != nil {
				return nil, err
			}
			var m Manifest
			if err := json.Unmarshal(raw, &m); err != nil {
				return nil, err
			}
			m.SourceDir = path
			m.ManifestPath = filepath.Join(path, "manifest.json")
			entry, err := pluginEntry(config, m.ID, false)
			if err != nil {
				return nil, err
			}
			m.Enabled = entry != nil
			for _, v := range array(config["disabledPlugins"]) {
				if v == m.ID {
					m.Enabled = false
				}
			}
			list = append(list, m)
		}
		return json.Marshal(list)
	}
}
func TestLocalPluginInstallRerunAndOwnership(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	dynamicPlugins(t, c)
	source := t.TempDir()
	writeFixture(t, filepath.Join(source, "manifest.json"), []byte(`{"id":"example.new","kinds":["overlay","bar-widget"]}`), 0600)
	writeFixture(t, filepath.Join(source, "View.qml"), []byte("import QtQuick\nItem {}\n"), 0600)
	ops, err := entryOperations(types.OmarchySetup{Name: "new", Plugins: []types.OmarchyPlugin{{ID: "example.new", Source: &types.OmarchySource{Path: source}, Enabled: ptr(true), Widget: &types.OmarchyWidget{Visible: ptr(false)}, Settings: map[string]any{"count": 5}}}}, "/blueprint.json")
	if err != nil {
		t.Fatal(err)
	}
	changes := 0
	apply := func() {
		t.Helper()
		if err := c.Apply(context.Background(), ops, func(r Result) {
			if r.Changed {
				changes++
			}
		}); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	first := changes
	calls := len(*commands)
	apply()
	if changes != first || len(*commands) != calls {
		t.Fatal("repeat run reinstalled or rewrote plugin")
	}
	dest := filepath.Join(c.Home, ".config/omarchy/plugins/example.new")
	if err := c.cleanOwned(context.Background(), "plugin/example.new", dest); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(dest, "View.qml"), []byte("edited locally"), 0600)
	if err := c.cleanOwned(context.Background(), "plugin/example.new", dest); err == nil {
		t.Fatal("modified plugin removable")
	}
}
func TestAllSourcesValidateBeforeInstall(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	dynamicPlugins(t, c)
	good, bad := t.TempDir(), t.TempDir()
	writeFixture(t, filepath.Join(good, "manifest.json"), []byte(`{"id":"example.a","kinds":["overlay"]}`), 0600)
	writeFixture(t, filepath.Join(bad, "manifest.json"), []byte(`{"id":"unexpected","kinds":["overlay"]}`), 0600)
	ops, err := entryOperations(types.OmarchySetup{Name: "sources", Plugins: []types.OmarchyPlugin{{ID: "example.a", Source: &types.OmarchySource{Path: good}}, {ID: "example.z", Source: &types.OmarchySource{Path: bad}}}}, "/blueprint.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Apply(context.Background(), ops, func(Result) {}); err == nil {
		t.Fatal("wrong manifest accepted")
	}
	if _, err := os.Stat(filepath.Join(c.Home, ".config/omarchy/plugins/example.a")); !os.IsNotExist(err) {
		t.Fatal("installed first source before validating last")
	}
	if len(*commands) != 0 {
		t.Fatal("mutated desktop before source validation")
	}
}
func TestSettingsMergeAndExplicitUnset(t *testing.T) {
	t.Parallel()
	cfg := &types.InitConfig{}
	raw := `{"configurations":[{"name":"a","tool":"omarchy","action":"set","plugins":[{"id":"example.overlay","settings":{"nested":{"one":false}}}]},{"name":"b","tool":"omarchy","action":"set","plugins":[{"id":"example.overlay","settings":{"nested":{"two":0}}}]}]}`
	ops, err := Load([]byte(raw), "json", "/blueprint.json", cfg)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := fixture(t)
	if err := c.Apply(context.Background(), ops, func(Result) {}); err != nil {
		t.Fatal(err)
	}
	unset, err := Load([]byte(`{"configurations":[{"name":"clear","tool":"omarchy","action":"set","plugins":[{"id":"example.overlay","unset":["nested.one"]}]}]}`), "json", "/blueprint.json", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Apply(context.Background(), unset, func(Result) {}); err != nil {
		t.Fatal(err)
	}
	config, _, err := c.config()
	if err != nil {
		t.Fatal(err)
	}
	p, err := pluginEntry(config, "example.overlay", false)
	if err != nil {
		t.Fatal(err)
	}
	nested := mapping(p["nested"])
	if _, exists := nested["one"]; exists || nested["two"] != float64(0) {
		t.Fatal("unset lost sibling")
	}
}
func TestPlacementConflictFailsBeforeWriting(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	before, err := c.read(".config/omarchy/shell.json")
	if err != nil {
		t.Fatal(err)
	}
	ops, err := entryOperations(types.OmarchySetup{Name: "bad anchors", Plugins: []types.OmarchyPlugin{{ID: "omarchy.menu", Widget: &types.OmarchyWidget{Before: "missing.widget"}}}}, "/blueprint.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Apply(context.Background(), ops, func(Result) {}); err == nil {
		t.Fatal("missing anchor accepted")
	}
	after, err := c.read(".config/omarchy/shell.json")
	if err != nil || string(before) != string(after) || len(*commands) != 0 {
		t.Fatal("failed placement mutated desktop")
	}
}
func TestExternalRouteActivationAndRestoration(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	c, _ := fixture(t)
	stock := filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver")
	writeFixture(t, stock, []byte("#!/bin/bash\nexit 0\n"), 0700)
	writeFixture(t, filepath.Join(c.Distribution, "shell/plugins/services/idle/Service.qml"), []byte("omarchy-launch-screensaver org.omarchy.screensaver screensaver-dismissed screensaverWindowCount"), 0600)
	original := "# preserved login settings\nexport PATH=" + shellQuote(filepath.Dir(stock)) + ":\"$PATH\"\n"
	writeFixture(t, filepath.Join(c.Home, ".bash_profile"), []byte(original), 0600)
	// Source the disposable login profile exactly as the login-shell boundary
	// does, without reading or editing the test operator's actual profile.
	c.Read = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "bash", "--noprofile", "--norc", "-c", `. "$1"; command -v omarchy-launch-screensaver`, "bash", filepath.Join(c.Home, ".bash_profile")).Output()
	}
	s := &types.OmarchyScreensaver{Mode: "external", Launch: &types.OmarchyExec{Exec: "/engine", Args: []string{"start"}}, Check: &types.OmarchyExec{Exec: "/engine", Args: []string{"check"}}}
	changed, err := c.screensaver(context.Background(), s)
	if err != nil || !changed {
		t.Fatalf("activate: %v", err)
	}
	changed, err = c.screensaver(context.Background(), s)
	if err != nil || changed {
		t.Fatalf("repeat changed route: %v", err)
	}
	changed, err = c.screensaver(context.Background(), &types.OmarchyScreensaver{Mode: "stock"})
	if err != nil || !changed {
		t.Fatalf("restore: %v", err)
	}
	profile, err := c.read(".bash_profile")
	if err != nil || string(profile) != original {
		t.Fatalf("profile did not restore: %q %v", profile, err)
	}
}
func TestHookFailureCannotLookLikeThemeSuccess(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	c, _ := fixture(t)
	source := filepath.Join(t.TempDir(), "hook")
	writeFixture(t, source, []byte("#!/bin/bash\nexit 23\n"), 0700)
	h := types.OmarchyHook{Event: "theme-set", Name: "20-test", Source: source}
	if _, err := c.hook(Operation{ID: "hook/theme-set/20-test", Kind: "hook", Hook: &h}); err != nil {
		t.Fatal(err)
	}
	c.hooks = []types.OmarchyHook{h}
	c.Run = func(cmd types.Command) error {
		run := exec.Command("bash", filepath.Join(c.Home, ".config/omarchy/hooks/theme-set.d/20-test"), "test")
		run.Env = append(os.Environ(), "RWR_OMARCHY_HOOK_RUN="+cmd.Variables["RWR_OMARCHY_HOOK_RUN"])
		_ = run.Run()
		return nil
	}
	if err := c.activateTheme("test"); err == nil {
		t.Fatal("native command hid hook failure")
	}
	writeFixture(t, source, []byte("#!/bin/bash\nexit 0\n"), 0700)
	if _, err := c.hook(Operation{ID: "hook/theme-set/20-test", Kind: "hook", Hook: &h}); err != nil {
		t.Fatal(err)
	}
	if err := c.activateTheme("test"); err != nil {
		t.Fatal(err)
	}
}
func TestMissingApplicationsAndInvalidIPCStopBeforeMutation(t *testing.T) {
	t.Parallel()
	c, commands := fixture(t)
	c.LookPath = func(name string) (string, error) {
		if name == "firefox" {
			return "", errors.New("missing")
		}
		return name, nil
	}
	ops := []Operation{{ID: "default/browser", Kind: "default", Path: []string{"browser"}, Value: "firefox"}}
	if err := c.Apply(context.Background(), ops, func(Result) {}); err == nil {
		t.Fatal("missing browser accepted")
	}
	if len(*commands) != 0 {
		t.Fatal("missing app launched installer")
	}
	c.Read = func(context.Context, string, ...string) ([]byte, error) { return nil, fmt.Errorf("no socket") }
	if err := c.Apply(context.Background(), ops, func(Result) {}); err == nil {
		t.Fatal("IPC failure treated as absence")
	}
}
