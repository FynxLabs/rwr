// Package diff compares the system scan against the resolved blueprint
// tree: what is on the machine that the tree does not declare, and what the
// tree declares that is gone. Its output is blueprint material - a list, a
// paste-ready block, or a routed edit - never an apply.
package diff

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/types"
)

// Change is one drift item.
type Change struct {
	Category string // packages, services, git, configs
	Provider string // packages only
	Name     string
	// Path is where the thing lives on the machine, for the categories that
	// are identified by one: a config's file and a checkout's directory. It
	// is what the journal is matched against, and what EmitBlocks needs to
	// find the scanned entry again. Empty for packages and services, which
	// are identified by name.
	Path    string
	Removal bool // declared in the tree, gone from the machine
}

// Machine is the scanned side, gathered once by the caller.
type Machine struct {
	Packages []scan.PackageResult
	Services []string
	Git      []scan.GitCheckout
	Configs  []scan.ConfigResult
	Home     string
}

// Compute joins machine, tree, and journal. A package the journal shows a
// run applied is not hand-added, whatever the current tree says - the tree
// may have changed since the apply.
func Compute(machine Machine, plan *types.Plan, applies []state.Entry) []Change {
	desired := map[string]map[string]bool{} // category → name set
	for _, resource := range plan.Resources {
		category := resource.Processor
		if desired[category] == nil {
			desired[category] = map[string]bool{}
		}
		desired[category][resource.Name] = true
	}
	applied := map[string]bool{} // "category\x00name" the journal accounts for
	// appliedPaths is the same question asked by location rather than by name,
	// which is the only way it can be answered for a config or a checkout.
	//
	// A files entry's blueprint name ("nvim-config") has nothing to do with the
	// file it lands on, so matching a scanned config against Identity["name"]
	// never hit: every dotfile rwr itself had applied came back as drift the
	// tree does not declare. The journal records where the thing actually went,
	// which is exactly what the scan found, so the two match on that.
	// status.Rows already enriches from the journal for the same reason.
	appliedPaths := map[string]bool{}
	for _, entry := range applies {
		applied[entry.Processor+"\x00"+entry.Identity["name"]] = true
		// dest for a file, target for a checkout: the recorded location.
		for _, key := range []string{"dest", "target"} {
			if location := entry.Identity[key]; location != "" {
				appliedPaths[entry.Processor+"\x00"+filepath.Clean(location)] = true
			}
		}
	}

	var changes []Change
	// add reports something found on the machine, unless the tree declares it
	// or the journal accounts for it. location is the absolute path for the
	// categories identified by one, and empty for the rest.
	add := func(category, provider, name, location string) {
		if desired[category][name] || applied[category+"\x00"+name] {
			return
		}
		if location != "" && appliedPaths[category+"\x00"+filepath.Clean(location)] {
			return
		}
		changes = append(changes, Change{Category: category, Provider: provider, Name: name, Path: location})
	}

	scannedPackages := map[string]bool{}
	for _, result := range machine.Packages {
		for _, name := range result.Names {
			scannedPackages[name] = true
			add(types.BlueprintTypePackages, result.Provider, name, "")
		}
	}
	for name := range desired[types.BlueprintTypePackages] {
		if !scannedPackages[name] {
			changes = append(changes, Change{Category: types.BlueprintTypePackages, Name: name, Removal: true})
		}
	}

	scannedServices := map[string]bool{}
	for _, service := range machine.Services {
		scannedServices[service] = true
		add(types.BlueprintTypeServices, "", service, "")
	}
	for name := range desired[types.BlueprintTypeServices] {
		if !scannedServices[name] {
			changes = append(changes, Change{Category: types.BlueprintTypeServices, Name: name, Removal: true})
		}
	}

	for _, checkout := range machine.Git {
		add(types.BlueprintTypeGit, "", path.Base(checkout.Path), checkout.Path)
	}

	for _, config := range machine.Configs {
		add(types.BlueprintTypeFiles, "", path.Base(config.Rel), config.Path)
	}

	sort.SliceStable(changes, func(i, j int) bool {
		a, b := changes[i], changes[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		return a.Name < b.Name
	})
	return changes
}

// Render formats changes as the readable grouped list.
func Render(changes []Change) string {
	if len(changes) == 0 {
		return "No drift: the machine matches the tree.\n"
	}
	var b strings.Builder
	lastGroup := ""
	for _, change := range changes {
		group := change.Category
		if change.Provider != "" {
			group += " (" + change.Provider + ")"
		}
		if group != lastGroup {
			fmt.Fprintf(&b, "%s:\n", group)
			lastGroup = group
		}
		marker := "+"
		if change.Removal {
			marker = "-"
		}
		fmt.Fprintf(&b, "  %s %s\n", marker, change.Name)
	}
	return b.String()
}

// EmitBlocks renders the additions as paste-ready blueprint blocks in the
// given format, one block per category present.
func EmitBlocks(changes []Change, machine Machine, format string) (string, error) {
	var out []string

	byProvider := map[string][]string{}
	var services []string
	var git []scan.GitCheckout
	var configs []scan.ConfigResult

	// Both lookups are by path, not by name. Two checkouts can share a
	// directory name and two configs very often share a filename - half a
	// dozen tools each own an "init.lua" or a "config" - so a name-keyed map
	// silently emits whichever one was scanned last.
	gitByPath := map[string]scan.GitCheckout{}
	for _, checkout := range machine.Git {
		gitByPath[filepath.Clean(checkout.Path)] = checkout
	}
	configByPath := map[string]scan.ConfigResult{}
	for _, config := range machine.Configs {
		configByPath[filepath.Clean(config.Path)] = config
	}

	for _, change := range changes {
		if change.Removal {
			continue
		}
		switch change.Category {
		case types.BlueprintTypePackages:
			byProvider[change.Provider] = append(byProvider[change.Provider], change.Name)
		case types.BlueprintTypeServices:
			services = append(services, change.Name)
		case types.BlueprintTypeGit:
			if checkout, ok := gitByPath[filepath.Clean(change.Path)]; ok {
				git = append(git, checkout)
			}
		case types.BlueprintTypeFiles:
			if config, ok := configByPath[filepath.Clean(change.Path)]; ok {
				configs = append(configs, config)
			}
		}
	}

	if len(byProvider) > 0 {
		providers := make([]string, 0, len(byProvider))
		for provider := range byProvider {
			providers = append(providers, provider)
		}
		sort.Strings(providers)
		var results []scan.PackageResult
		for _, provider := range providers {
			results = append(results, scan.PackageResult{Provider: provider, Names: byProvider[provider]})
		}
		block, err := scan.EmitPackages(results, format)
		if err != nil {
			return "", err
		}
		out = append(out, string(block))
	}
	if len(services) > 0 {
		block, err := scan.EmitServices(services, format)
		if err != nil {
			return "", err
		}
		out = append(out, string(block))
	}
	if len(git) > 0 {
		block, err := scan.EmitGit(git, machine.Home, format)
		if err != nil {
			return "", err
		}
		out = append(out, string(block))
	}
	// Configs were listed as drift and then dropped here, so `rwr diff --emit`
	// printed blocks for everything except the dotfiles - the category the
	// command is most often reached for. scan.EmitConfigs already existed;
	// capture has been using it all along.
	if len(configs) > 0 {
		block, err := scan.EmitConfigs(configs, machine.Home, format)
		if err != nil {
			return "", err
		}
		out = append(out, string(block))
	}
	return strings.Join(out, "\n"), nil
}
