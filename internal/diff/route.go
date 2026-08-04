package diff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
)

// categoryDirs maps a change category to the tree directories its blueprints
// conventionally live in.
var categoryDirs = map[string][]string{
	types.BlueprintTypePackages: {"packages"},
	types.BlueprintTypeServices: {"services"},
	types.BlueprintTypeGit:      {"git"},
	types.BlueprintTypeFiles:    {"files"},
}

// Destinations discovers where a category's entries can land: the tree's own
// blueprint files for that category, plus any file those blueprints import
// (the Common files a shared tree pulls in). rwr cannot know whether a change
// is machine-specific or shared; it can only surface the real candidates.
func Destinations(tree, category string) ([]string, error) {
	seen := map[string]bool{}
	var destinations []string
	for _, dir := range categoryDirs[category] {
		entries, err := os.ReadDir(filepath.Join(tree, dir))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !helpers.IsBlueprintFile(entry.Name()) {
				continue
			}
			path := filepath.Join(tree, dir, entry.Name())
			if !seen[path] {
				seen[path] = true
				destinations = append(destinations, path)
			}
			for _, imported := range importTargets(path) {
				if !seen[imported] {
					seen[imported] = true
					destinations = append(destinations, imported)
				}
			}
		}
	}
	sort.Strings(destinations)
	if len(destinations) == 0 {
		return nil, fmt.Errorf("no %s blueprint files found under %s", category, tree)
	}
	return destinations, nil
}

// importTargets resolves the files a blueprint imports, file-relative.
func importTargets(path string) []string {
	data, err := os.ReadFile(path) // #nosec G304 -- operator's own blueprint tree
	if err != nil {
		return nil
	}
	format, err := helpers.FormatForPath(path)
	if err != nil {
		return nil
	}
	var doc map[string]interface{}
	if err := helpers.UnmarshalBlueprint(data, format, &doc); err != nil {
		return nil
	}
	var targets []string
	for _, value := range doc {
		entries, ok := value.([]interface{})
		if !ok {
			continue
		}
		for _, raw := range entries {
			entry, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if imported, ok := entry["import"].(string); ok && imported != "" {
				resolved := filepath.Join(filepath.Dir(path), imported)
				if _, err := os.Stat(resolved); err == nil {
					targets = append(targets, resolved)
				}
			}
		}
	}
	return targets
}

// AppendEntries appends blueprint entries under the category's key in the
// destination file, in that file's own format, and rewrites it.
func AppendEntries(destination, category string, entries []map[string]interface{}) error {
	data, err := os.ReadFile(destination) // #nosec G304 -- operator-chosen destination inside their own tree
	if err != nil {
		return err
	}
	format, err := helpers.FormatForPath(destination)
	if err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := helpers.UnmarshalBlueprint(data, format, &doc); err != nil {
		return fmt.Errorf("%s does not parse: %w", destination, err)
	}

	key := categoryKey(category)
	var existing []interface{}
	if current, ok := doc[key].([]interface{}); ok {
		existing = current
	}
	for _, entry := range entries {
		existing = append(existing, entry)
	}
	doc[key] = existing

	encoded, err := helpers.EncodeBlueprintDoc(doc, format)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, encoded, 0o644) // #nosec G306 -- blueprint files are world-readable config
}

// categoryKey is the top-level blueprint key for a category.
func categoryKey(category string) string {
	if category == types.BlueprintTypeSSHKeys {
		return "ssh_keys"
	}
	return strings.ToLower(category)
}

// PackageEntries builds the entries AppendEntries takes for a provider's
// added packages.
func PackageEntries(provider string, names []string) []map[string]interface{} {
	return []map[string]interface{}{{
		"names":           names,
		"action":          "install",
		"package_manager": provider,
	}}
}

// ServiceEntries builds service enable entries.
func ServiceEntries(names []string) []map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		entries = append(entries, map[string]interface{}{"name": name, "action": "enable"})
	}
	return entries
}
