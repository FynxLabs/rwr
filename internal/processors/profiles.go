package processors

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/fynxlabs/rwr/internal/types"
)

// ProfileSummary is what a tree's blueprints declare about profiles.
type ProfileSummary struct {
	// Names are the profile names found, sorted.
	Names []string
	// Counts is the number of entries carrying each profile.
	Counts map[string]int
	// BaseItems is the number of entries carrying no profile, which always apply.
	BaseItems int
	// Files is the number of blueprint files inspected.
	Files int
}

// CollectProfiles walks the blueprint tree and reports the profiles it declares.
//
// This reads the blueprints. `rwr profiles` used to read only the arrays written
// inline in the init file, and profiles are declared on blueprint entries - so the
// command that exists to tell an operator what `--profile` accepts answered "No
// profiles found" for every tree that uses profiles.
func CollectProfiles(initConfig *types.InitConfig) (*ProfileSummary, error) {
	summary := &ProfileSummary{Counts: map[string]int{}}

	location := initConfig.Init.Location
	if location == "" {
		return finish(summary), nil
	}
	files, err := GetBlueprintFileOrder(location, initConfig.Init.Order, initConfig.Init.RunOnlyListed, initConfig)
	if err != nil {
		return nil, err
	}
	type profileFile struct{ path, processor, section string }
	seen := map[profileFile]bool{}
	active := map[profileFile]bool{}
	var visit func(string, string, string) error
	visit = func(path, processor, section string) error {
		absolute, err := filepath.Abs(path)
		path = absolute
		if err != nil {
			return err
		}
		key := profileFile{path, processor, section}
		if active[key] {
			return fmt.Errorf("circular profile import at %s", path)
		}
		if seen[key] {
			return nil
		}
		active[key] = true
		defer delete(active, key)
		top, _, err := decodeTopLevel(path, initConfig)
		if err != nil {
			return fmt.Errorf("reading profiles from %s: %w", path, err)
		}
		for name, value := range top {
			if section != "" && name != section {
				continue
			}
			if processor != types.BlueprintTypeBootstrap && blueprintKeyToType[name] != processor {
				continue
			}
			if _, known := blueprintKeyToType[name]; !known {
				continue
			}
			if section == "" {
				if err := visit(path, blueprintKeyToType[name], name); err != nil {
					return err
				}
				continue
			}
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}
			var entries []struct {
				Profiles []string `json:"profiles"`
				Import   string   `json:"import"`
			}
			if err := json.Unmarshal(data, &entries); err != nil {
				return fmt.Errorf("profiles in %s (%s): %w", path, name, err)
			}
			for _, entry := range entries {
				if entry.Import != "" {
					if err := visit(filepath.Join(filepath.Dir(path), entry.Import), blueprintKeyToType[name], name); err != nil {
						return err
					}
					continue
				}
				if len(entry.Profiles) == 0 {
					summary.BaseItems++
				}
				for _, profile := range entry.Profiles {
					if profile != "" {
						summary.Counts[profile]++
					}
				}
			}
		}
		seen[key] = true
		return nil
	}
	for processor, paths := range files {
		for _, path := range paths {
			if err := visit(filepath.Join(location, path), processor, ""); err != nil {
				return nil, err
			}
		}
	}
	if path := findBootstrapFile(location); path != "" {
		if err := visit(path, types.BlueprintTypeBootstrap, ""); err != nil {
			return nil, err
		}
	}
	uniqueFiles := map[string]bool{}
	for key := range seen {
		uniqueFiles[key.path] = true
	}
	summary.Files = len(uniqueFiles)

	return finish(summary), nil
}

func finish(summary *ProfileSummary) *ProfileSummary {
	for name := range summary.Counts {
		summary.Names = append(summary.Names, name)
	}
	sort.Strings(summary.Names)
	return summary
}
