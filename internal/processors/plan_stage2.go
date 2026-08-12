package processors

import (
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// ResolveStage2 completes the Plan after init and bootstrap: provider states
// and the resource enumeration the lanes count. It cannot run earlier -
// bootstrap can install the package manager later blueprints depend on, so
// detecting providers before it produces wrong lanes.
//
// osInfo picks the default provider for entries that do not pin a
// package_manager (nil falls back to the alphabetically first available).
// The plan must name the provider the executor will actually use: keying
// unpinned entries on "" put them in an "items" lane that no runtime
// LaneUpdate (which keys on the resolved provider) ever touched, so the
// checklist showed a full brew bar and a ghost pending lane forever.
func ResolveStage2(plan *types.Plan, osInfo *types.OSInfo) {
	available := system.GetAvailableProviders()
	for name, provider := range available {
		plan.Providers = append(plan.Providers, types.ProviderState{
			Name:      name,
			Available: true,
			Elevated:  provider.Elevated,
		})
	}

	defaultProvider := ""
	if provider, ok := defaultProviderFor(osInfo, available); ok {
		defaultProvider = provider.Name
	}

	for processor, files := range plan.Files {
		for _, file := range files {
			plan.Resources = append(plan.Resources, enumerateResources(processor, file, defaultProvider)...)
		}
	}
}

// enumerateResources lists the planned units of work one resolved file
// declares. Decode failures return nothing here: stage 1 already reported
// them as diagnostics, and stage 2 must not duplicate the noise.
// defaultProvider fills in for package/repository entries that do not pin a
// package_manager, matching what the executor will resolve at run time.
func enumerateResources(processor string, file types.ResolvedFile, defaultProvider string) []types.Resource {
	usesProviders := processor == types.BlueprintTypePackages || processor == types.BlueprintTypeRepositories
	var resources []types.Resource
	add := func(provider, name, action string) {
		if name == "" {
			return
		}
		if provider == "" && usesProviders {
			provider = defaultProvider
		}
		resources = append(resources, types.Resource{
			Processor: processor,
			Provider:  provider,
			Name:      name,
			Action:    action,
			Status:    types.StatusPlanned,
		})
	}

	switch processor {
	case types.BlueprintTypePackages:
		var d types.PackagesData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, pkg := range d.Packages {
			names := pkg.Names
			if len(names) == 0 && pkg.Name != "" {
				names = []string{pkg.Name}
			}
			for _, name := range names {
				add(pkg.PackageManager, name, pkg.Action)
			}
		}
	case types.BlueprintTypeRepositories:
		var d types.RepositoriesData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, repo := range d.Repositories {
			add(repo.PackageManager, repo.Name, repo.Action)
		}
	case types.BlueprintTypeFiles:
		var d types.FileData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, f := range d.Files {
			names := f.Names
			if len(names) == 0 {
				names = []string{f.Name}
			}
			for _, name := range names {
				add("", name, f.Action)
			}
		}
		for _, tmpl := range d.Templates {
			add("", tmpl.Name, "template")
		}
		for _, dir := range d.Directories {
			names := dir.Names
			if len(names) == 0 {
				names = []string{dir.Name}
			}
			for _, name := range names {
				add("", name, dir.Action)
			}
		}
	case types.BlueprintTypeServices:
		var d types.ServiceData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, svc := range d.Services {
			add("", svc.Name, svc.Action)
		}
	case types.BlueprintTypeGit:
		var d types.GitData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, repo := range d.Repos {
			add("", repo.Name, repo.Action)
		}
	case types.BlueprintTypeScripts:
		var d types.ScriptData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, script := range d.Scripts {
			add("", script.Name, script.Action)
		}
	case types.BlueprintTypeSSHKeys:
		var d types.SSHKeyData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, key := range d.SSHKeys {
			add("", key.Name, "ssh_key")
		}
	case types.BlueprintTypeFonts:
		var d types.FontsData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, font := range d.Fonts {
			names := font.Names
			if len(names) == 0 && font.Name != "" {
				names = []string{font.Name}
			}
			for _, name := range names {
				add(font.Provider, name, font.Action)
			}
		}
	case types.BlueprintTypeUsers:
		var d types.UsersData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, user := range d.Users {
			add("", user.Name, user.Action)
		}
		for _, group := range d.Groups {
			add("", group.Name, group.Action)
		}
	case types.BlueprintTypeConfiguration:
		var d types.ConfigData
		if helpers.DecodeBlueprintInto(file.Resolved, file.Format, processor, 0, &d) != nil {
			return nil
		}
		for _, cfg := range d.Configurations {
			names := cfg.Names
			if len(names) == 0 && cfg.Name != "" {
				names = []string{cfg.Name}
			}
			for _, name := range names {
				add("", name, "configure")
			}
		}
	}
	return resources
}
