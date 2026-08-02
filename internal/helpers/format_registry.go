package helpers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/types"
)

// The format registry is the single source of blueprint format knowledge:
// which extensions exist, which format a file is, and what candidate names
// init/bootstrap discovery should try. Before it existed this knowledge was
// duplicated across ~15 sites, each a hand edit for every new format — and the
// copies had already diverged (half the tree assumed one format per tree via
// Init.Format, half derived per file).
//
// Formats are canonical names ("yaml", "json", "toml"); extensions map onto
// them, so ".yml" and ".yaml" are both format "yaml".
var formatByExtension = map[string]string{
	types.FormatExtYAML:    types.FormatYAML,
	types.FormatExtYAMLAlt: types.FormatYAML,
	types.FormatExtJSON:    types.FormatJSON,
	types.FormatExtTOML:    types.FormatTOML,
	types.FormatExtCUE:     types.FormatCUE,
}

// extensionOrder is the preference order for discovery and error messages:
// yaml first because it is what the docs lead with and most trees use, so
// probing (init.<ext> over the network) usually hits on the first try.
var extensionOrder = []string{
	types.FormatExtYAML,
	types.FormatExtYAMLAlt,
	types.FormatExtJSON,
	types.FormatExtTOML,
	types.FormatExtCUE,
}

// KnownExtensions returns every recognized blueprint extension (with the
// leading dot), in preference order.
func KnownExtensions() []string {
	return append([]string(nil), extensionOrder...)
}

// FormatForPath resolves a file's format from its extension. An extensionless
// or unknown-extension path is a diagnostic error naming the path — not a
// panic, which is what `filepath.Ext(file)[1:]` produced for an extensionless
// file.
func FormatForPath(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "", fmt.Errorf("%s has no extension; blueprint files need one of %s", path, strings.Join(KnownExtensions(), ", "))
	}
	format, ok := formatByExtension[ext]
	if !ok {
		return "", fmt.Errorf("%s: unsupported blueprint format %q; supported extensions are %s", path, ext, strings.Join(KnownExtensions(), ", "))
	}
	return format, nil
}

// IsBlueprintFile reports whether the path carries a recognized blueprint
// extension. It is the walk-time filter: files that fail it are simply not
// blueprints (READMEs, scripts), which is not an error.
func IsBlueprintFile(path string) bool {
	_, ok := formatByExtension[strings.ToLower(filepath.Ext(path))]
	return ok
}

// CandidateFilenames returns the discovery candidates for a base name
// ("init", "bootstrap") — one per known extension, in stable order.
func CandidateFilenames(base string) []string {
	exts := KnownExtensions()
	names := make([]string, 0, len(exts))
	for _, ext := range exts {
		names = append(names, base+ext)
	}
	return names
}

// CanonicalFormat maps any accepted format spelling — canonical name, alias
// ("yml"), or dotted extension (".yaml") — to the canonical format name.
func CanonicalFormat(format string) (string, error) {
	f := strings.ToLower(strings.TrimSpace(format))
	if canonical, ok := formatByExtension["."+strings.TrimPrefix(f, ".")]; ok {
		return canonical, nil
	}
	return "", fmt.Errorf("unsupported blueprint format %q; supported formats are yaml, json, toml", format)
}
