package helpers

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// ImportPath represents a single import directive
type ImportPath struct {
	Path string `mapstructure:"import" yaml:"import" json:"import" toml:"import"`
}

// ProcessImports reads and merges imported blueprint files
// blueprintDir: the directory containing the current blueprint file
// imports: list of import paths (relative to blueprintDir)
// format: the format of the files (yaml, json, toml)
// target: pointer to the target struct to unmarshal into
func ProcessImports(blueprintDir string, imports []ImportPath, format string, target interface{}) error {
	if len(imports) == 0 {
		return nil
	}

	log.Debugf("Processing %d import(s)", len(imports))

	for _, imp := range imports {
		if imp.Path == "" {
			log.Warn("Skipping empty import path")
			continue
		}

		// Resolve the import path relative to the blueprint directory
		importPath := filepath.Join(blueprintDir, imp.Path)
		log.Debugf("Processing import: %s", importPath)

		// Read the import file
		data, err := os.ReadFile(importPath)
		if err != nil {
			return fmt.Errorf("error reading import file %s: %w", importPath, err)
		}

		// Determine format from file extension if not explicitly provided
		fileFormat := format
		if fileFormat == "" {
			ext := filepath.Ext(importPath)
			fileFormat = ext
		}

		// Unmarshal the imported data into the target
		if err := UnmarshalBlueprint(data, fileFormat, target); err != nil {
			return fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
		}

		log.Debugf("Successfully processed import: %s", importPath)
	}

	return nil
}

// ProcessImportsRecursive processes imports that may themselves contain imports
// This prevents infinite recursion by tracking visited files
func ProcessImportsRecursive(blueprintDir string, imports []ImportPath, format string, target interface{}, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}

	for _, imp := range imports {
		if imp.Path == "" {
			continue
		}

		// Resolve absolute path to detect circular imports
		importPath := filepath.Join(blueprintDir, imp.Path)
		absPath, err := filepath.Abs(importPath)
		if err != nil {
			return fmt.Errorf("error resolving import path %s: %w", importPath, err)
		}

		// Check for circular import
		if visited[absPath] {
			log.Warnf("Circular import detected, skipping: %s", absPath)
			continue
		}
		visited[absPath] = true

		// Read and process the import
		data, err := os.ReadFile(importPath)
		if err != nil {
			return fmt.Errorf("error reading import file %s: %w", importPath, err)
		}

		fileFormat := format
		if fileFormat == "" {
			ext := filepath.Ext(importPath)
			fileFormat = ext
		}

		if err := UnmarshalBlueprint(data, fileFormat, target); err != nil {
			return fmt.Errorf("error unmarshaling import file %s: %w", importPath, err)
		}

		log.Debugf("Successfully processed import: %s", importPath)
	}

	return nil
}
