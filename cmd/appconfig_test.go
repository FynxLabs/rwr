package cmd

import "testing"

// Two trees, two AppConfigs, zero cross-talk: parsing flags on one must not
// leak into the other. Impossible before the refactor - seventeen package
// globals meant every test shared (and had to reset) the same state.
func TestAppConfig_InstancesAreIsolated(t *testing.T) {
	appA, appB := NewAppConfig(), NewAppConfig()
	rootA, rootB := NewRootCmd(appA), NewRootCmd(appB)

	if err := rootA.PersistentFlags().Parse([]string{"--debug", "--dry-run", "--profile", "work", "--init-file", "a.yaml"}); err != nil {
		t.Fatalf("parsing A's flags: %v", err)
	}
	if err := rootB.PersistentFlags().Parse([]string{"--log-level", "error"}); err != nil {
		t.Fatalf("parsing B's flags: %v", err)
	}

	if !appA.Debug || !appA.DryRun || appA.InitFilePath != "a.yaml" || len(appA.Profiles) != 1 {
		t.Errorf("A did not receive its own flags: %+v", appA)
	}
	if appB.Debug || appB.DryRun || appB.InitFilePath != "" || len(appB.Profiles) != 0 {
		t.Errorf("B received A's flags: %+v", appB)
	}
	if appB.LogLevel != "error" {
		t.Errorf("B.LogLevel = %q, want error", appB.LogLevel)
	}
	if appA.LogLevel == "error" {
		t.Error("A received B's log level")
	}

	// Defaults hold on a fresh instance regardless of what other trees did.
	if c := NewAppConfig(); !c.Interactive || c.LogLevel != "info" {
		t.Errorf("NewAppConfig defaults = %+v", c)
	}
}
