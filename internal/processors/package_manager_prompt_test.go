package processors

import (
	"errors"
	"testing"

	"github.com/freehold-digital/rwr/internal/exectest"
	"github.com/freehold-digital/rwr/internal/reporting"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

type cancelledManagerChoice struct {
	err     error
	prompts int
}

func (*cancelledManagerChoice) SupportsInlinePrompts() bool { return true }
func (r *cancelledManagerChoice) Emit(event reporting.Event) {
	if req, ok := event.(reporting.FormReq); ok {
		r.prompts++
		if reporting.TryClaim(req.Claim) {
			req.Done <- r.err
		}
	}
}

func TestCancelledManagerChoiceDoesNotInstallDefault(t *testing.T) {
	for _, promptErr := range []error{reporting.ErrPromptCancelled, reporting.ErrPromptUnavailable} {
		t.Run(promptErr.Error(), func(t *testing.T) {
			reporter := &cancelledManagerChoice{err: promptErr}
			defer reporting.Set(reporter)()
			recorder := exectest.New()
			defer system.SetExecutor(recorder)()
			defer system.SetProvidersForTest(map[string]*types.Provider{
				"brew": {Name: "brew", Detection: types.DetectionConfig{Binary: "rwr-test-missing-brew"},
					Install: types.InstallConfig{Steps: []types.ActionStep{{Action: "command", Exec: "must-not-install"}}}},
			})()
			config := &types.InitConfig{}
			config.Variables.Flags.Interactive = true
			osInfo := &types.OSInfo{}
			osInfo.System.OS = types.OSDarwin
			err := preparePackageManagers(nil, osInfo, config, true)
			if !errors.Is(err, promptErr) {
				t.Fatalf("selection error = %v, want %v", err, promptErr)
			}
			if reporter.prompts != 1 {
				t.Fatalf("prompts = %d, want 1", reporter.prompts)
			}
			if len(recorder.Calls) != 0 {
				t.Fatalf("canceled choice ran installation: %+v", recorder.Calls)
			}
		})
	}
}
