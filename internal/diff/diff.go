// Package diff compares the system scan against the resolved blueprint
// tree: what is on the machine that the tree does not declare, and what the
// tree declares that is gone. Its output is blueprint material - a list, a
// paste-ready block, or a routed edit - never an apply.
package diff

import (
	"fmt"
	"path"
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
	Removal  bool // declared in the tree, gone from the machine
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
	for _, entry := range applies {
		applied[entry.Processor+"\x00"+entry.Identity["name"]] = true
	}

	var changes []Change
	add := func(category, provider, name string) {
		if desired[category][name] || applied[category+"\x00"+name] {
			return
		}
		changes = append(changes, Change{Category: category, Provider: provider, Name: name})
	}

	scannedPackages := map[string]bool{}
	for _, result := range machine.Packages {
		for _, name := range result.Names {
			scannedPackages[name] = true
			add(types.BlueprintTypePackages, result.Provider, name)
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
		add(types.BlueprintTypeServices, "", service)
	}
	for name := range desired[types.BlueprintTypeServices] {
		if !scannedServices[name] {
			changes = append(changes, Change{Category: types.BlueprintTypeServices, Name: name, Removal: true})
		}
	}

	for _, checkout := range machine.Git {
		add(types.BlueprintTypeGit, "", path.Base(checkout.Path))
	}

	for _, config := range machine.Configs {
		add(types.BlueprintTypeFiles, "", path.Base(config.Rel))
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
	gitByName := map[string]scan.GitCheckout{}
	for _, checkout := range machine.Git {
		gitByName[path.Base(checkout.Path)] = checkout
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
			if checkout, ok := gitByName[change.Name]; ok {
				git = append(git, checkout)
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
	return strings.Join(out, "\n"), nil
}
