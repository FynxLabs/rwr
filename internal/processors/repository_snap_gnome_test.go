package processors

import (
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// The shipped snap provider gates its last add step on HasInterfaces, which was
// absent from the predicates map - an underivable predicate is a hard error, so
// every snap repository add failed at that step, interface or no interface.
func TestProcessRepositories_SnapAddWithoutInterface(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)
	provider := providerForTest(t, "snap", sourcesDir, keysDir)

	rec := exectest.New()
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"snap": provider})()

	if err := processRepositories([]types.Repository{{
		Name:           "bitwarden",
		PackageManager: "snap",
		Action:         "add",
		URL:            "https://snapcraft.io/bitwarden",
	}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
		t.Fatalf("processRepositories: %v", err)
	}

	calls := rec.Find("snap")
	if len(calls) != 1 {
		t.Fatalf("recorded %d snap calls, want 1 (install only): %v", len(calls), rec.Calls)
	}
	want := []string{"snap", "install", "bitwarden"}
	if got := calls[0].Argv(); !equalStrings(got, want) {
		t.Errorf("snap argv = %#v, want %#v", got, want)
	}
	assertNoPlaceholders(t, rec)
}

// interface/slot on the blueprint drive the connect step after the install.
func TestProcessRepositories_SnapAddConnectsInterface(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	for _, tt := range []struct {
		name        string
		slot        string
		wantConnect []string
	}{
		{
			name:        "named slot",
			slot:        "core:home",
			wantConnect: []string{"snap", "connect", "bitwarden:password-manager-service", "core:home"},
		},
		{
			name:        "system slot",
			slot:        "",
			wantConnect: []string{"snap", "connect", "bitwarden:password-manager-service"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := providerForTest(t, "snap", sourcesDir, keysDir)

			rec := exectest.New()
			defer system.SetExecutor(rec)()
			defer system.SetProvidersForTest(map[string]*types.Provider{"snap": provider})()

			if err := processRepositories([]types.Repository{{
				Name:           "bitwarden",
				PackageManager: "snap",
				Action:         "add",
				URL:            "https://snapcraft.io/bitwarden",
				Interface:      "password-manager-service",
				Slot:           tt.slot,
			}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
				t.Fatalf("processRepositories: %v", err)
			}

			calls := rec.Find("snap")
			if len(calls) != 2 {
				t.Fatalf("recorded %d snap calls, want install + connect: %v", len(calls), rec.Calls)
			}
			if got := calls[1].Argv(); !equalStrings(got, tt.wantConnect) {
				t.Errorf("connect argv = %#v, want %#v", got, tt.wantConnect)
			}
			assertNoPlaceholders(t, rec)
		})
	}
}

// The shipped gnome-extensions provider gates its last remove step on
// ResetSettings, which no blueprint field carried - so every extension remove
// failed at the reset step.
func TestProcessRepositories_GnomeExtensionsRemove(t *testing.T) {
	sourcesDir, keysDir := repoDirs(t)

	for _, tt := range []struct {
		name      string
		reset     bool
		wantCalls int
	}{
		{name: "keep settings", reset: false, wantCalls: 2},
		{name: "reset settings", reset: true, wantCalls: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := providerForTest(t, "gnome-extensions", sourcesDir, keysDir)

			rec := exectest.New()
			defer system.SetExecutor(rec)()
			defer system.SetProvidersForTest(map[string]*types.Provider{"gnome-extensions": provider})()

			if err := processRepositories([]types.Repository{{
				Name:           "dash-to-dock",
				PackageManager: "gnome-extensions",
				Action:         "remove",
				UUID:           "dash-to-dock@micxgx.gmail.com",
				ResetSettings:  tt.reset,
			}}, &types.OSInfo{}, &types.InitConfig{}); err != nil {
				t.Fatalf("processRepositories: %v", err)
			}

			calls := rec.Find("gnome-extensions")
			if len(calls) != tt.wantCalls {
				t.Fatalf("recorded %d calls, want %d: %v", len(calls), tt.wantCalls, rec.Calls)
			}
			if tt.reset {
				want := []string{"gnome-extensions", "reset", "dash-to-dock@micxgx.gmail.com"}
				if got := calls[2].Argv(); !equalStrings(got, want) {
					t.Errorf("reset argv = %#v, want %#v", got, want)
				}
			}
			assertNoPlaceholders(t, rec)
		})
	}
}
