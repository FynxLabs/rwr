package processors

import (
	"errors"
	"runtime"
	"testing"

	"github.com/freehold-digital/rwr/internal/exectest"
	"github.com/freehold-digital/rwr/internal/system"
	"github.com/freehold-digital/rwr/internal/types"
)

func TestPackageFailureDoesNotRetryOnRawTerminal(t *testing.T) {
	rec := exectest.New()
	rec.Err = errors.New("download failed: HTTP 404")
	defer system.SetExecutor(rec)()
	defer system.SetProvidersForTest(map[string]*types.Provider{"fixture": {Name: "fixture", Escalates: true, Detection: types.DetectionConfig{Binary: "sh", Distributions: []string{runtime.GOOS}}, Commands: types.CommandConfig{Install: "install"}}})()
	_ = ProcessPackages([]byte("packages:\n  - name: fixture-package\n    action: install\n    package_manager: fixture\n"), nil, t.TempDir(), "yaml", &types.OSInfo{}, &types.InitConfig{})
	if len(rec.Calls) != 1 {
		t.Fatalf("command calls=%d, want one captured attempt", len(rec.Calls))
	}
	for _, call := range rec.Calls {
		if call.Interactive {
			t.Error("unrelated HTTP failure retried with raw terminal access")
		}
	}
}
