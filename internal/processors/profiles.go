package processors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/helpers"
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
// inline in the init file, and profiles are declared on blueprint entries — so the
// command that exists to tell an operator what `--profile` accepts answered "No
// profiles found" for every tree that uses profiles.
func CollectProfiles(initConfig *types.InitConfig) (*ProfileSummary, error) {
	summary := &ProfileSummary{Counts: map[string]int{}}

	// The init file may carry entries of its own; keep counting those.
	collectFrom(summary, initConfig.Packages)
	collectFrom(summary, initConfig.Services)
	collectFrom(summary, initConfig.Files)
	collectFrom(summary, initConfig.Templates)
	collectFrom(summary, initConfig.Directories)
	collectFrom(summary, initConfig.Repositories)

	location := initConfig.Init.Location
	if location == "" {
		return finish(summary), nil
	}

	err := filepath.WalkDir(location, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != location && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Per-file via the registry: filtering on the tree-wide Init.Format
		// made profile discovery blind to every blueprint in another format.
		if !helpers.IsBlueprintFile(path) {
			return nil
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if base == "init" {
			return nil
		}

		blueprintType := blueprintTypeForPath(location, path)
		if blueprintType == "" {
			return nil
		}

		data, err := os.ReadFile(path) // #nosec G304 G122 -- read-only walk of the operator's own blueprint tree; containment added in PR8
		if err != nil {
			log.Warnf("Could not read %s: %v", path, err)
			return nil
		}

		resolved, err := helpers.ResolveTemplateForValidation(data, initConfig.Variables)
		if err != nil {
			log.Warnf("Could not resolve variables in %s: %v", path, err)
			return nil
		}

		format, formatErr := helpers.FormatForPath(path)
		if formatErr != nil {
			log.Warnf("Could not determine format of %s: %v", path, formatErr)
			return nil
		}
		if err := collectFromFile(summary, resolved, format, blueprintType); err != nil {
			log.Warnf("Could not read profiles from %s: %v", path, err)
			return nil
		}
		summary.Files++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking blueprint tree %s: %w", location, err)
	}

	return finish(summary), nil
}

// blueprintTypeForPath names the blueprint type from the directory a file sits in,
// the same way the run order does.
func blueprintTypeForPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		switch part {
		case types.BlueprintTypePackages, types.BlueprintTypeRepositories, types.BlueprintTypeFiles,
			types.BlueprintTypeServices, types.BlueprintTypeUsers, types.BlueprintTypeGit,
			types.BlueprintTypeScripts, types.BlueprintTypeSSHKeys, types.BlueprintTypeFonts,
			types.BlueprintTypeConfiguration:
			return part
		}
	}
	return ""
}

// collectFromFile decodes one blueprint and records the profiles its entries carry.
func collectFromFile(summary *ProfileSummary, data []byte, format, blueprintType string) error {
	switch blueprintType {
	case types.BlueprintTypePackages:
		var d types.PackagesData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Packages)
	case types.BlueprintTypeRepositories:
		var d types.RepositoriesData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Repositories)
	case types.BlueprintTypeFiles:
		var d types.FileData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Files)
		collectFrom(summary, d.Templates)
		collectFrom(summary, d.Directories)
	case types.BlueprintTypeServices:
		var d types.ServiceData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Services)
	case types.BlueprintTypeGit:
		var d types.GitData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Repos)
	case types.BlueprintTypeScripts:
		var d types.ScriptData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Scripts)
	case types.BlueprintTypeSSHKeys:
		var d types.SSHKeyData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.SSHKeys)
	case types.BlueprintTypeUsers:
		var d types.UsersData
		if err := helpers.DecodeBlueprintInto(data, format, blueprintType, 0, &d); err != nil {
			return err
		}
		collectFrom(summary, d.Users)
		collectFrom(summary, d.Groups)
	}
	return nil
}

// collectFrom records one slice of profile-carrying entries.
func collectFrom[T interface{ GetProfiles() []string }](summary *ProfileSummary, items []T) {
	for _, item := range items {
		profiles := item.GetProfiles()
		if len(profiles) == 0 {
			summary.BaseItems++
			continue
		}
		for _, profile := range profiles {
			if profile == "" {
				continue
			}
			summary.Counts[profile]++
		}
	}
}

func finish(summary *ProfileSummary) *ProfileSummary {
	for name := range summary.Counts {
		summary.Names = append(summary.Names, name)
	}
	sort.Strings(summary.Names)
	return summary
}
