package status

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/display"
	"github.com/fynxlabs/rwr/internal/state"
	"github.com/fynxlabs/rwr/internal/system"
	"github.com/fynxlabs/rwr/internal/types"
)

// Class is one drift verdict.
type Class string

const (
	InSync       Class = "in-sync"
	Missing      Class = "missing"
	ModifiedItem Class = "modified"
	UnknownItem  Class = "unknown"
	Stale        Class = "stale" // recorded, no longer in the tree
)

// Row is one desired-or-recorded unit with its verdict.
type Row struct {
	Processor string
	Name      string
	Class     Class
	Note      string
}

// Rows joins the plan's desired resources with the journal and the queries.
// The journal enriches identity (a plan resource has no dest path; its
// entry does); without one, classes the queries cannot honestly decide are
// unknown.
func Rows(plan *types.Plan, applies []state.Entry, querier *Querier) []Row {
	// Identity enrichment uses every recorded apply - a reversal does not
	// erase where an identity lives on disk; stale detection uses only the
	// unreversed ones - deliberately removed work is not stale.
	recorded := map[string]*state.Entry{}
	unreversed := map[string]*state.Entry{}
	for i := range applies {
		entry := &applies[i]
		key := entryStatusKey(*entry)
		recorded[key] = entry
		if !entry.Reversed {
			unreversed[key] = entry
		}
	}

	var rows []Row
	seen := map[string]bool{}
	for _, resource := range plan.Resources {
		key := resourceStatusKey(resource)
		seen[key] = true
		rows = append(rows, classify(resource, recorded[key], querier))
	}

	// Recorded but no longer desired: stale, the class only a record makes
	// possible.
	staleKeys := make([]string, 0, len(unreversed))
	for key := range unreversed {
		if !seen[key] {
			staleKeys = append(staleKeys, key)
		}
	}
	sort.Strings(staleKeys)
	for _, key := range staleKeys {
		entry := unreversed[key]
		rows = append(rows, Row{
			Processor: entry.Processor,
			Name:      entry.Identity["name"],
			Class:     Stale,
			Note:      "recorded by a past run, absent from the tree",
		})
	}
	return rows
}

// Files and git checkouts are identified by where they live, not by a display
// name that can legitimately repeat. Packages, services, and the remaining
// locationless resources retain name matching.
func resourceStatusKey(resource types.Resource) string {
	if resource.Location != "" && (resource.Processor == types.BlueprintTypeFiles || resource.Processor == types.BlueprintTypeGit) {
		return resource.Processor + "\x00path\x00" + filepath.Clean(resource.Location)
	}
	return resource.Processor + "\x00name\x00" + resource.Name
}

func entryStatusKey(entry state.Entry) string {
	location := ""
	switch entry.Processor {
	case types.BlueprintTypeFiles:
		location = entry.Identity["dest"]
	case types.BlueprintTypeGit:
		location = entry.Identity["target"]
	}
	if location != "" {
		return entry.Processor + "\x00path\x00" + filepath.Clean(location)
	}
	return entry.Processor + "\x00name\x00" + entry.Identity["name"]
}

// classify decides one desired resource's verdict.
func classify(resource types.Resource, entry *state.Entry, querier *Querier) Row {
	row := Row{Processor: resource.Processor, Name: resource.Name}

	switch resource.Processor {
	case types.BlueprintTypePackages:
		provider, ok := system.GetProvider(providerFor(resource, entry))
		if !ok {
			row.Class, row.Note = UnknownItem, "provider not available"
			return row
		}
		switch querier.PackagePresent(provider, resource.Name) {
		case Present:
			row.Class = InSync
		case Absent:
			row.Class = Missing
		default:
			row.Class, row.Note = UnknownItem, "no usable list query"
		}
	case types.BlueprintTypeFiles:
		if entry == nil || entry.Identity["dest"] == "" {
			row.Class, row.Note = UnknownItem, "no recorded destination"
			return row
		}
		switch FileState(entry.Identity["dest"], entry.Identity["sha256"]) {
		case Present:
			row.Class = InSync
		case Absent:
			row.Class = Missing
		case Modified:
			row.Class, row.Note = ModifiedItem, "content differs from the recorded apply"
		default:
			row.Class = UnknownItem
		}
	case types.BlueprintTypeServices:
		switch ServiceState(resource.Name) {
		case Present:
			row.Class = InSync
		case Absent:
			row.Class = Missing
		default:
			row.Class, row.Note = UnknownItem, "no platform query"
		}
	case types.BlueprintTypeGit:
		if entry == nil || entry.Identity["target"] == "" {
			row.Class, row.Note = UnknownItem, "no recorded checkout target"
			return row
		}
		if PathPresent(entry.Identity["target"]) == Present {
			row.Class = InSync
		} else {
			row.Class = Missing
		}
	default:
		// scripts, configuration, users, ssh_keys, repositories, fonts: a
		// query that cannot be honest is worse than none.
		row.Class, row.Note = UnknownItem, "not queryable"
	}
	return row
}

func providerFor(resource types.Resource, entry *state.Entry) string {
	if resource.Provider != "" {
		return resource.Provider
	}
	if entry != nil {
		return entry.Identity["provider"]
	}
	return ""
}

// Render formats rows as the drift table; hasRecord toggles the stale
// explanation footer.
func Render(rows []Row, hasRecord bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %s %s\n", display.PadRight("PROCESSOR", 14), display.PadRight("NAME", 40), display.PadRight("STATE", 10), "NOTE")
	for _, row := range rows {
		fmt.Fprintf(&b, "%s %s %s %s\n",
			display.PadRight(row.Processor, 14),
			display.PadRight(display.Truncate(row.Name, 40), 40),
			display.PadRight(string(row.Class), 10),
			row.Note,
		)
	}
	if !hasRecord {
		b.WriteString("\n(no run record: recorded-identity checks unavailable - run `rwr all` once to establish one)\n")
	}
	return b.String()
}

// Drifted reports whether any row demands attention (exit code 1).
func Drifted(rows []Row) bool {
	for _, row := range rows {
		switch row.Class {
		case Missing, ModifiedItem, Stale:
			return true
		}
	}
	return false
}
