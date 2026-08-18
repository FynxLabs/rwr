package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"charm.land/log/v2"
	"github.com/fynxlabs/rwr/internal/types"
)

// importVariables are the template variables imported blueprint files resolve
// against. The run entry points set them once (the executor seam pattern);
// without this, an imported file's `{{ .User.home }}` stayed literal - the
// top-level file went through ResolveTemplate in the run loop, but imports
// were read and decoded raw, and rwr created a directory literally named
// "{{ .User.home }}" in the working directory.
var (
	importVarsMu    sync.RWMutex
	importVariables *types.Variables
)

// SetTemplateVariables installs the variables imports resolve against
// (nil clears). Returns a restore func for tests.
func SetTemplateVariables(vars *types.Variables) (restore func()) {
	importVarsMu.Lock()
	previous := importVariables
	importVariables = vars
	importVarsMu.Unlock()
	return func() {
		importVarsMu.Lock()
		importVariables = previous
		importVarsMu.Unlock()
	}
}

func templateVariables() *types.Variables {
	importVarsMu.RLock()
	defer importVarsMu.RUnlock()
	return importVariables
}

// ResolveImports expands `import:` entries into the items they name, following
// imports that the imported files declare in turn.
//
// Every blueprint type carried its own copy of this loop - ten of them - and none
// recursed: an imported file's own imports were decoded into a struct and then
// dropped, so a shared file that imports a base file contributed nothing from the
// base. Each copy also carried cycle detection that could never fire, because it
// tracked visited paths within one level of a walk that never went deeper.
//
// itemImport reads the import path off an item; decode reads a blueprint file of
// this type and returns its items. Both are supplied by the caller because the
// item and data types differ per blueprint type; everything else is common.
func ResolveImports[T any](
	items []T,
	blueprintDir string,
	itemImport func(T) string,
	decode func(data []byte, format string) ([]T, error),
	format string,
) ([]T, error) {
	return resolveImports(items, blueprintDir, itemImport, decode, format, map[string]bool{})
}

func resolveImports[T any](
	items []T,
	blueprintDir string,
	itemImport func(T) string,
	decode func(data []byte, format string) ([]T, error),
	format string,
	visited map[string]bool,
) ([]T, error) {
	resolved := make([]T, 0, len(items))

	for _, item := range items {
		importPath := itemImport(item)
		if importPath == "" {
			resolved = append(resolved, item)
			continue
		}

		fullPath := filepath.Join(blueprintDir, importPath)
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			return nil, fmt.Errorf("error resolving import path %s: %w", fullPath, err)
		}

		// A cycle is a mistake in the blueprints, and continuing would either loop
		// or silently apply an arbitrary subset. Say so.
		if visited[absPath] {
			return nil, fmt.Errorf("circular import: %s imports itself, directly or through another file", absPath)
		}
		visited[absPath] = true

		data, err := os.ReadFile(fullPath) // #nosec G304 -- path is inside the operator's own blueprint tree
		if err != nil {
			return nil, fmt.Errorf("error reading import file %s: %w", fullPath, err)
		}

		// Imported files get the same template resolution the importing file
		// got in the run loop - `{{ .User.home }}` in an imported blueprint
		// must not survive to execution as a literal path.
		if vars := templateVariables(); vars != nil {
			resolved, resolveErr := ResolveTemplate(data, *vars)
			if resolveErr != nil {
				return nil, fmt.Errorf("error resolving variables in import %s: %w", fullPath, resolveErr)
			}
			data = resolved
		}

		// The imported file's own extension decides its format; the importing
		// file's format is only the fallback for a file with no recognized
		// extension. It used to be the other way around: the parent's format
		// was forced onto every import, so a .toml file imported from a .yaml
		// blueprint was fed to the YAML decoder.
		fileFormat, formatErr := FormatForPath(fullPath)
		if formatErr != nil {
			fileFormat = format
		}

		imported, err := decode(data, fileFormat)
		if err != nil {
			return nil, fmt.Errorf("error reading import file %s: %w", fullPath, err)
		}

		// The imported file's own imports resolve relative to the directory that
		// file lives in, not the directory that imported it.
		nested, err := resolveImports(imported, filepath.Dir(fullPath), itemImport, decode, format, visited)
		if err != nil {
			return nil, err
		}

		log.Debugf("Imported %d item(s) from %s", len(nested), importPath)
		resolved = append(resolved, nested...)

		// An import is one edge of the graph, not a permanent mark: the same shared
		// file reached from two different branches is not a cycle. Only a path on
		// the current chain counts.
		delete(visited, absPath)
	}

	return resolved, nil
}
