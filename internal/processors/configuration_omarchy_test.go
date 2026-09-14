package processors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/exectest"
	"github.com/fynxlabs/rwr/internal/reporting"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

type omarchyReviewReporter struct{ emit func(reporting.Event) }

func (r omarchyReviewReporter) Emit(e reporting.Event) { r.emit(e) }

func TestOmarchyRoutesThroughConfigurationProcessor(t *testing.T) {
	t.Parallel()
	tree := t.TempDir()
	raw := `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","shell":{"idle":{"lock":300}}}]}`
	if err := os.WriteFile(filepath.Join(tree, "desktop.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &types.InitConfig{Init: types.Init{Location: tree, Format: "json"}}
	plan, err := ResolveStage1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := Stage1Error(plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 1 || len(plan.Files[types.BlueprintTypeConfiguration]) != 1 {
		t.Fatalf("omarchy configuration routed as %+v", plan.FileOrder)
	}
	if _, exists := plan.Files["omarchy"]; exists {
		t.Fatal("standalone omarchy processor bucket exists")
	}
	ResolveStage2(plan, &types.OSInfo{})
	if len(plan.Resources) != 1 {
		t.Fatalf("resources = %+v", plan.Resources)
	}
	resource := plan.Resources[0]
	if resource.Processor != types.BlueprintTypeConfiguration || resource.Provider != "omarchy" || resource.Name != "shell/idle/lock" {
		t.Fatalf("omarchy resource escaped configuration provider: %+v", resource)
	}
}
func TestStandaloneOmarchyBlueprintIsRejected(t *testing.T) {
	t.Parallel()
	tree := t.TempDir()
	raw := `{"omarchy":[{"name":"desktop","shell":{"idle":{"lock":300}}}]}`
	if err := os.WriteFile(filepath.Join(tree, "desktop.json"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &types.InitConfig{Init: types.Init{Location: tree, Format: "json"}}
	plan, err := ResolveStage1(cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = Stage1Error(plan)
	if err == nil || !strings.Contains(err.Error(), `unknown field "omarchy"`) {
		t.Fatalf("standalone omarchy blueprint was not rejected clearly: %v", err)
	}
}

func TestConfigurationPreflightsOmarchyAcrossFilesBeforeOtherTools(t *testing.T) {
	recorder := exectest.New()
	defer system.SetExecutor(recorder)()

	files := []types.ResolvedFile{
		{
			Path:      "/blueprints/first.json",
			Processor: types.BlueprintTypeConfiguration,
			Format:    "json",
			Resolved:  []byte(`{"configurations":[{"name":"dock","tool":"macos_defaults","action":"set","key":"orientation","kind":"string","value":"right"},{"name":"first","tool":"omarchy","action":"set","theme":{"name":"one","active":true}}]}`),
		},
		{
			Path:      "/blueprints/second.json",
			Processor: types.BlueprintTypeConfiguration,
			Format:    "json",
			Resolved:  []byte(`{"configurations":[{"name":"second","tool":"omarchy","action":"set","theme":{"name":"two","active":true}}]}`),
		},
	}

	err := ProcessConfigurationFiles(files, &types.InitConfig{})
	if err == nil || !strings.Contains(err.Error(), "theme/active") {
		t.Fatalf("cross-file Omarchy conflict was not rejected: %v", err)
	}
	if len(recorder.Calls) != 0 {
		t.Fatalf("ordinary configuration ran before Omarchy preflight: %+v", recorder.Calls)
	}
}

func TestAllOmarchyConfigurationErrorDecisions(t *testing.T) {
	// These cases intentionally change the process-wide reporter and dry-run
	// flag. Keep them sequential and use an unavailable desktop in a temp HOME.
	for _, tt := range []struct {
		name                 string
		interactive          bool
		decision             reporting.HaltDecision
		wantError, wantLater bool
	}{
		{"retry", true, reporting.HaltRetry, false, true},
		{"skip", true, reporting.HaltSkip, true, true},
		{"abort", true, reporting.HaltAbort, true, false},
		{"headless", false, reporting.HaltAbort, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home, tree := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
			system.SetDryRun(false)
			t.Cleanup(func() { system.SetDryRun(false); resetFailures() })
			for path, content := range map[string]string{
				"configuration.json": `{"configurations":[{"name":"desktop","tool":"omarchy","action":"set","shell":{"idle":{"lock":300}}}]}`,
				"files.json":         `{"files":[]}`,
			} {
				if err := os.WriteFile(filepath.Join(tree, path), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cfg := &types.InitConfig{Init: types.Init{Location: tree, Format: "json"}}
			cfg.Variables.UserDefined = map[string]interface{}{}
			cfg.Variables.Flags.Interactive = tt.interactive
			halts, finished, planned := 0, 0, 0
			later := false
			var finishErr error
			var runErrors []types.StepError
			restore := reporting.Set(omarchyReviewReporter{emit: func(event reporting.Event) {
				switch e := event.(type) {
				case reporting.HaltReq:
					halts++
					if finished != 0 || later {
						t.Error("processor finished or continued before halt decision")
					}
					if !e.Retryable || e.Processor != types.BlueprintTypeConfiguration {
						t.Errorf("wrong halt: %+v", e)
					}
					// The retry runs the actual processor's dry-run path, yielding
					// planned resources and avoiding live desktop writes.
					if tt.decision == reporting.HaltRetry {
						system.SetDryRun(true)
					}
					if reporting.TryClaim(e.Claim) {
						e.Decision <- tt.decision
					}
				case reporting.ProcFinished:
					if e.Processor == types.BlueprintTypeConfiguration {
						finished++
						finishErr = e.Err
					}
				case reporting.ProcStarted:
					if e.Processor == types.BlueprintTypeFiles {
						later = true
					}
				case reporting.ResourceDone:
					if e.Resource.Processor == types.BlueprintTypeConfiguration && e.Resource.Provider == "omarchy" && e.Resource.Status == types.StatusPlanned {
						planned++
					}
				case reporting.RunFinished:
					runErrors = e.Errs
				}
			}})
			defer restore()
			err := All(cfg, &types.OSInfo{}, []string{types.BlueprintTypeConfiguration, types.BlueprintTypeFiles})
			if (err != nil) != tt.wantError {
				t.Fatalf("run error = %v", err)
			}
			if finished != 1 || (finishErr != nil) != tt.wantError {
				t.Fatalf("completion count=%d error=%v", finished, finishErr)
			}
			if later != tt.wantLater {
				t.Fatalf("later processor ran=%t", later)
			}
			wantHalts := 0
			if tt.interactive {
				wantHalts = 1
			}
			if halts != wantHalts {
				t.Fatalf("halt count=%d", halts)
			}
			if tt.decision == reporting.HaltRetry && (planned != 1 || failureCount() != 0 || len(runErrors) != 0) {
				t.Fatalf("retry: planned=%d failures=%d errors=%v", planned, failureCount(), runErrors)
			}
			if tt.wantLater && tt.wantError && len(runErrors) != 1 {
				t.Fatalf("final step errors=%v", runErrors)
			}
		})
	}
}
