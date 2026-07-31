package processors

import (
	"slices"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// Every processor named in the default run order must have a dispatch case in
// All. An entry without one falls through to `default:` and logs "Unknown
// processor", so a blueprint of that type is silently skipped — which is what
// packageManagers did.
func TestDefaultRunOrder_EveryEntryIsDispatched(t *testing.T) {
	dispatched := []string{
		types.BlueprintTypeRepositories,
		types.BlueprintTypePackages,
		types.BlueprintTypeFiles,
		types.BlueprintTypeServices,
		types.BlueprintTypeUsers,
		types.BlueprintTypeGit,
		types.BlueprintTypeScripts,
		types.BlueprintTypeSSHKeys,
		types.BlueprintTypeFonts,
		types.BlueprintTypeConfiguration,
	}

	for _, processor := range defaultRunOrder {
		if !slices.Contains(dispatched, processor) {
			t.Errorf("default run order includes %q, which All has no dispatch case for; "+
				"blueprints of that type would be silently skipped", processor)
		}
	}
}

// users was missing from the default order, so `rwr all` never processed user
// blueprints unless the init file hand-wrote its own order — even though
// `rwr run users` worked, which made the gap easy to miss.
func TestDefaultRunOrder_IncludesEveryDispatchableProcessor(t *testing.T) {
	// Every type that has a dispatch case in All and can appear as a blueprint
	// directory must be reachable from a default `rwr all`.
	want := []string{
		types.BlueprintTypeRepositories,
		types.BlueprintTypePackages,
		types.BlueprintTypeFiles,
		types.BlueprintTypeServices,
		types.BlueprintTypeUsers,
		types.BlueprintTypeGit,
		types.BlueprintTypeScripts,
		types.BlueprintTypeSSHKeys,
		types.BlueprintTypeFonts,
		types.BlueprintTypeConfiguration,
	}

	for _, processor := range want {
		if !slices.Contains(defaultRunOrder, processor) {
			t.Errorf("%q has a dispatch case but is missing from the default run order, "+
				"so `rwr all` would never run it", processor)
		}
	}
}

// packageManagers runs ahead of the blueprint loop, from initConfig, and has no
// case in the dispatch switch. Listing it in the run order only produced an
// "Unknown processor" warning.
func TestDefaultRunOrder_ExcludesPackageManagers(t *testing.T) {
	if slices.Contains(defaultRunOrder, types.BlueprintTypePackageManagers) {
		t.Error("packageManagers must not be in the blueprint run order; it is processed " +
			"separately from initConfig before the loop and has no dispatch case")
	}
}

func TestGetBlueprintRunOrder_UsesDefaultWhenInitSpecifiesNone(t *testing.T) {
	order, err := GetBlueprintRunOrder(&types.InitConfig{})
	if err != nil {
		t.Fatalf("GetBlueprintRunOrder: %v", err)
	}

	if !slices.Equal(order, defaultRunOrder) {
		t.Errorf("run order = %v, want the default %v", order, defaultRunOrder)
	}
}

func TestGetBlueprintRunOrder_HonoursInitOrder(t *testing.T) {
	initConfig := &types.InitConfig{}
	initConfig.Init.Order = []interface{}{"packages", "files"}

	order, err := GetBlueprintRunOrder(initConfig)
	if err != nil {
		t.Fatalf("GetBlueprintRunOrder: %v", err)
	}

	want := []string{"packages", "files"}
	if !slices.Equal(order, want) {
		t.Errorf("run order = %v, want %v", order, want)
	}
}
