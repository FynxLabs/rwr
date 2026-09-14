package omarchy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestRouteResolutionFailureRestoresPreviousFiles(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	writeFixture(t, filepath.Join(c.Home, ".bash_profile"), []byte("# original\n"), 0600)
	writeFixture(t, filepath.Join(c.Distribution, "shell/plugins/services/idle/Service.qml"), []byte("omarchy-launch-screensaver org.omarchy.screensaver screensaver-dismissed screensaverWindowCount"), 0600)
	writeFixture(t, filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver"), []byte("#!/bin/bash\n"), 0700)
	c.Read = func(context.Context, string, ...string) ([]byte, error) { return []byte("/unrelated/launcher"), nil }
	_, err := c.screensaver(context.Background(), &types.OmarchyScreensaver{Mode: "external", Launch: &types.OmarchyExec{Exec: "/engine"}, Check: &types.OmarchyExec{Exec: "/engine"}})
	if err == nil {
		t.Fatal("accepted wrong launcher")
	}
	profile, err := c.read(".bash_profile")
	if err != nil || string(profile) != "# original\n" {
		t.Fatal("profile was not restored")
	}
	if _, err := c.read(routePath); !os.IsNotExist(err) {
		t.Fatal("failed route remains installed")
	}
}
func TestModifiedHookPayloadIsRetained(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	source := filepath.Join(t.TempDir(), "source")
	writeFixture(t, source, []byte("original"), 0700)
	h := types.OmarchyHook{Event: "theme-set", Name: "test", Source: source}
	op := Operation{ID: "hook/theme-set/test", Kind: "hook", Hook: &h}
	if _, err := c.hook(op); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(c.Home, hookPayload(h)), []byte("local edit"), 0700)
	if _, err := c.hook(op); err == nil {
		t.Fatal("modified payload overwritten")
	}
}
func TestExplicitStockEnableWithDisabledCloneIsAllowed(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	snap, err := c.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	snap.Plugins["example.clone"] = Manifest{ID: "example.clone", Clone: "omarchy.workspaces", Enabled: true, Kinds: []string{"bar-widget"}}
	ops, err := entryOperations(types.OmarchySetup{Name: "restore stock", Plugins: []types.OmarchyPlugin{{ID: "example.clone", Enabled: ptr(false)}, {ID: "omarchy.workspaces", Enabled: ptr(true)}}}, "/blueprint.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Preflight(context.Background(), ops, snap); err != nil {
		t.Fatal(err)
	}
	ops, err = entryOperations(types.OmarchySetup{Name: "conflict", Plugins: []types.OmarchyPlugin{{ID: "example.clone", Enabled: ptr(true)}, {ID: "omarchy.workspaces", Enabled: ptr(true)}}}, "/blueprint.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Preflight(context.Background(), ops, snap); err == nil {
		t.Fatal("conflicting stock and clone enabled")
	}
}
func TestThemeOverlayConvergesAndModifiedAssetsSurvive(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	source := t.TempDir()
	writeFixture(t, filepath.Join(source, "shell.toml"), []byte("opacity = 0.9\n"), 0600)
	writeFixture(t, filepath.Join(c.Distribution, "themes/stock/colors.toml"), []byte("accent = '#fff'\n"), 0600)
	op := Operation{ID: "theme/stock", Kind: "theme-source", Theme: &types.OmarchyTheme{Name: "stock", Source: &types.OmarchySource{Path: source}}}
	changed, err := c.themeSource(context.Background(), op)
	if err != nil || !changed {
		t.Fatalf("install overlay: %v", err)
	}
	changed, err = c.themeSource(context.Background(), op)
	if err != nil || changed {
		t.Fatalf("repeat overlay: %v", err)
	}
	target := filepath.Join(c.Home, ".config/omarchy/themes/stock/shell.toml")
	writeFixture(t, target, []byte("user edit"), 0600)
	if _, err := c.themeSource(context.Background(), op); err == nil {
		t.Fatal("overwrote modified overlay")
	}
	raw, err := os.ReadFile(target)
	if err != nil || string(raw) != "user edit" {
		t.Fatal("lost overlay edit")
	}
}
func TestAllResourcesGetOutcomeAfterLostIPC(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	c.Run = func(types.Command) error {
		c.Read = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("lost IPC") }
		return nil
	}
	ops := []Operation{{ID: "plugin/example.overlay/enabled", Kind: "enabled", Plugin: &types.OmarchyPlugin{ID: "example.overlay", Enabled: ptr(false)}}, {ID: "default/editor", Kind: "default", Path: []string{"editor"}, Value: "nvim"}}
	count := 0
	if err := c.Apply(context.Background(), ops, func(Result) { count++ }); err == nil {
		t.Fatal("lost IPC reported success")
	}
	if count != len(ops) {
		t.Fatalf("only %d resources terminated", count)
	}
}

func TestScreensaverRequiresStockIdleRoute(t *testing.T) {
	t.Parallel()
	c, _ := fixture(t)
	snap, err := c.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	snap.Plugins["omarchy.idle"] = Manifest{ID: "omarchy.idle", FirstParty: true, Enabled: false, Kinds: []string{"service"}}
	snap.Plugins["personal.idle"] = Manifest{ID: "personal.idle", Clone: "omarchy.idle", Enabled: true, Kinds: []string{"service"}}
	screen := Operation{ID: "integration/screensaver", Kind: "screensaver", Screensaver: &types.OmarchyScreensaver{Mode: "external"}}
	if err := c.Preflight(context.Background(), []Operation{screen}, snap); err == nil {
		t.Fatal("active clone treated as stock route")
	}
	restore := Operation{ID: "plugin/omarchy.idle/enabled", Kind: "enabled", Plugin: &types.OmarchyPlugin{ID: "omarchy.idle", Enabled: ptr(true)}}
	if err := c.Preflight(context.Background(), []Operation{restore, screen}, snap); err == nil {
		t.Fatal("stock enable without explicit clone disable accepted")
	}
	disable := Operation{ID: "plugin/personal.idle/enabled", Kind: "enabled", Plugin: &types.OmarchyPlugin{ID: "personal.idle", Enabled: ptr(false)}}
	if err := c.Preflight(context.Background(), []Operation{restore, disable, screen}, snap); err != nil {
		t.Fatal(err)
	}
}

func TestScreensaverOwnershipFailureRollsBackActivation(t *testing.T) {
	t.Parallel()
	for _, prior := range []bool{false, true} {
		for _, committed := range []bool{false, true} {
			t.Run(fmt.Sprintf("prior=%t/committed=%t", prior, committed), func(t *testing.T) {
				t.Parallel()
				c, _ := fixture(t)
				writeFixture(t, filepath.Join(c.Distribution, "shell/plugins/services/idle/Service.qml"), []byte("omarchy-launch-screensaver org.omarchy.screensaver screensaver-dismissed screensaverWindowCount"), 0600)
				writeFixture(t, filepath.Join(c.Distribution, "bin/omarchy-launch-screensaver"), []byte("#!/bin/bash\n"), 0700)
				c.Read = func(context.Context, string, ...string) ([]byte, error) {
					return []byte(filepath.Join(c.Home, routePath)), nil
				}
				var oldRoute, oldProfile, oldOwnership []byte
				if prior {
					oldRoute = []byte("#!/bin/bash\n# prior engine\n")
					oldProfile = append([]byte("# personal profile\n"), c.routeBlock()...)
					oldOwnership = []byte(fmt.Sprintf("{\n  %q: {\"source\":\".bash_profile\",\"hash\":%q},\n  \"plugin/other\": {\"source\":\"keep\",\"hash\":\"keep\"}\n}\n", "integration/screensaver", hash(oldRoute)))
					writeFixture(t, filepath.Join(c.Home, routePath), oldRoute, 0700)
					writeFixture(t, filepath.Join(c.Home, ".bash_profile"), oldProfile, 0600)
					writeFixture(t, filepath.Join(c.Home, ownershipPath), oldOwnership, 0600)
				}
				injected := errors.New("ownership persistence failed")
				failed := false
				c.replaceFile = func(path string, old, data []byte, mode fs.FileMode) (bool, error) {
					if path == ownershipPath && !failed {
						failed = true
						if committed {
							changed, err := c.replaceOnDisk(path, old, data, mode)
							return changed, errors.Join(injected, err)
						}
						return false, injected
					}
					return c.replaceOnDisk(path, old, data, mode)
				}
				saver := &types.OmarchyScreensaver{Mode: "external", Launch: &types.OmarchyExec{Exec: "/engine"}, Check: &types.OmarchyExec{Exec: "/engine"}}
				if _, err := c.screensaver(context.Background(), saver); !errors.Is(err, injected) {
					t.Fatalf("activation error = %v", err)
				}
				for path, want := range map[string][]byte{routePath: oldRoute, ".bash_profile": oldProfile, ownershipPath: oldOwnership} {
					got, err := c.read(path)
					if want == nil {
						if !errors.Is(err, fs.ErrNotExist) {
							t.Fatalf("new file %s remains: %v", path, err)
						}
					} else if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("%s not restored: %q, %v", path, got, err)
					}
				}
				// A fresh attempt must converge after the storage problem is fixed.
				c.replaceFile = nil
				if _, err := c.screensaver(context.Background(), saver); err != nil {
					t.Fatal(err)
				}
				if changed, err := c.screensaver(context.Background(), saver); err != nil || changed {
					t.Fatalf("repeat apply: changed=%t err=%v", changed, err)
				}
			})
		}
	}
}
