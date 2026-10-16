package system

import (
	"testing"

	"github.com/freehold-digital/rwr/internal/exectest"
	"github.com/freehold-digital/rwr/internal/types"
)

func TestStartupDependenciesPreserveNestedSudoMetadata(t *testing.T) {
	previous := corePackageTool
	corePackageTool = func(string) types.ToolInfo { return types.ToolInfo{} }
	t.Cleanup(func() { corePackageTool = previous })
	provider := &types.Provider{Name: "fixture", BinPath: "fixture", Escalates: true, Detection: types.DetectionConfig{Binary: "sh"}, Commands: types.CommandConfig{Install: "install"}, CorePackages: map[string][]string{"openssl": {"fixture-ssl"}, "build-essentials": {"fixture-build"}}}
	defer SetProvidersForTest(map[string]*types.Provider{"fixture": provider})()
	rec := exectest.New()
	defer SetExecutor(rec)()
	info := &types.OSInfo{}
	info.PackageManager.Default.Name = "fixture"
	for _, install := range []func(*types.OSInfo, *types.InitConfig) error{InstallOpenSSL, InstallBuildEssentials} {
		if err := install(info, &types.InitConfig{}); err != nil {
			t.Fatal(err)
		}
	}
	if len(rec.Calls) != 2 {
		t.Fatalf("calls=%d", len(rec.Calls))
	}
	for _, call := range rec.Calls {
		if !call.Escalates || call.Elevated || call.Interactive {
			t.Fatalf("lost managed escalation policy: %+v", call)
		}
	}
	if !GetPackageManagerInfo(provider, "fixture").Escalates {
		t.Fatal("cleanup metadata lost nested sudo flag")
	}
}
