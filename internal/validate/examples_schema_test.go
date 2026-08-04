package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/types"
	"gopkg.in/yaml.v3"
)

// blueprintTargets maps a blueprint directory name to the type its files decode
// into. This is the contract the examples are asserted against.
func blueprintTarget(kind string) interface{} {
	switch kind {
	case types.BlueprintTypePackages:
		return &types.PackagesData{}
	case types.BlueprintTypeRepositories:
		return &types.RepositoriesData{}
	case types.BlueprintTypeFiles:
		return &types.FileData{}
	case types.BlueprintTypeServices:
		return &types.ServiceData{}
	case types.BlueprintTypeGit:
		return &types.GitData{}
	case types.BlueprintTypeScripts:
		return &types.ScriptData{}
	case types.BlueprintTypeSSHKeys:
		return &types.SSHKeyData{}
	case types.BlueprintTypeFonts:
		return &types.FontsData{}
	case types.BlueprintTypeUsers:
		return &types.UsersData{}
	case types.BlueprintTypeConfiguration:
		return &types.ConfigData{}
	default:
		return nil
	}
}

// multiTypeBlueprint is the target for an example that carries several blueprint
// types in one file (examples/alternative_layouts/minimal_files/all_in_one.*).
// RWR has no single struct for that layout, so the fields here are exactly the
// per-type fields from the structs above; the file is asserted against the union
// of the real schemas rather than being skipped.
type multiTypeBlueprint struct {
	types.SchemaVersion `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`
	Packages            []types.Package       `yaml:"packages,omitempty" json:"packages,omitempty" toml:"packages,omitempty"`
	Repositories        []types.Repository    `yaml:"repositories,omitempty" json:"repositories,omitempty" toml:"repositories,omitempty"`
	Files               []types.File          `yaml:"files,omitempty" json:"files,omitempty" toml:"files,omitempty"`
	Templates           []types.File          `yaml:"templates,omitempty" json:"templates,omitempty" toml:"templates,omitempty"`
	Directories         []types.Directory     `yaml:"directories,omitempty" json:"directories,omitempty" toml:"directories,omitempty"`
	Services            []types.Service       `yaml:"services,omitempty" json:"services,omitempty" toml:"services,omitempty"`
	Git                 []types.Git           `yaml:"git,omitempty" json:"git,omitempty" toml:"git,omitempty"`
	Scripts             []types.Script        `yaml:"scripts,omitempty" json:"scripts,omitempty" toml:"scripts,omitempty"`
	SSHKeys             []types.SSHKey        `yaml:"ssh_keys,omitempty" json:"ssh_keys,omitempty" toml:"ssh_keys,omitempty"`
	Fonts               []types.Font          `yaml:"fonts,omitempty" json:"fonts,omitempty" toml:"fonts,omitempty"`
	Groups              []types.Group         `yaml:"groups,omitempty" json:"groups,omitempty" toml:"groups,omitempty"`
	Users               []types.User          `yaml:"users,omitempty" json:"users,omitempty" toml:"users,omitempty"`
	Configurations      []types.Configuration `yaml:"configurations,omitempty" json:"configurations,omitempty" toml:"configurations,omitempty"`
}

// blueprintTargetForPath decides which struct an example file must decode into,
// from its path relative to examples/.
//
// This deliberately mirrors production: getProcessorType (processors/blueprints.go)
// scans *every* path segment for a blueprint type name, it does not look only at
// the immediate parent directory. Using the parent alone - as this test used to -
// gave a nil target to every init and bootstrap file (their parent is the format
// name, "yaml"/"json"/"toml"), to everything under alternative_layouts/, and to
// examples/imports/Common/packages/arch/base-aur.yaml, all of which were then
// silently skipped by the very check that is supposed to cover them.
func blueprintTargetForPath(rel string) interface{} {
	base := filepath.Base(rel)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	// An init file configures the run; a bootstrap file is its own type. Neither
	// lives in a directory named after its type.
	switch stem {
	case "init":
		return &types.InitConfig{}
	case types.BlueprintTypeBootstrap:
		return &types.BootstrapData{}
	case "manifest":
		// A multi-configuration repo root file, not a blueprint.
		return &types.Manifest{}
	}

	// Same rule production uses: the first path segment naming a blueprint type wins.
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if target := blueprintTarget(part); target != nil {
			return target
		}
	}

	// Flattened layouts (examples/alternative_layouts/flattened) keep one blueprint
	// type per file at the tree root, so the file name is what names the type.
	if target := blueprintTarget(stem); target != nil {
		return target
	}

	// A single file holding several blueprint types (all_in_one.*).
	return &multiTypeBlueprint{}
}

// knownSchemaViolations records example files that do NOT decode cleanly, and the
// single unknown key each one is allowed to carry.
//
// It is currently empty, and that is the goal state. Widening the type derivation
// above brought init, bootstrap, alternative_layouts/ and imports/ under the strict
// check for the first time and uncovered two real bugs - `asUser` on script entries
// and `variables.userDefined` in init files - both of which are now implemented
// rather than tolerated.
//
// This is a shrinking list, never a growing one: an entry whose file starts
// decoding cleanly fails the test, so fixing an example forces its removal.
var knownSchemaViolations = map[string]struct{ key string }{}

// Every example must decode into its blueprint struct with no unknown fields.
//
// This is the backwards-compatibility check: the decoders RWR uses in production
// ignore unknown keys, so renaming or removing a struct field silently turns
// every blueprint using it into a no-op rather than an error. Decoding strictly
// here means such a change fails the build with the field named, instead of
// shipping and quietly doing nothing on users' machines.
//
// It also catches the reverse - an example inventing a field that never existed,
// which is how `src`/`dest`, `username` and script `mode` survived in the tree.
func TestExamples_DecodeStrictlyIntoBlueprintStructs(t *testing.T) {
	root := examplesDir(t)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		switch ext {
		case "yaml", "yml", "json", "toml":
		default:
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		target := blueprintTargetForPath(rel)

		raw, readErr := os.ReadFile(path) // #nosec G304 -- test walks the repo's own examples tree
		if readErr != nil {
			t.Errorf("%s: %v", path, readErr)
			return nil
		}
		resolved, tmplErr := helpers.ResolveTemplate(raw, exampleVariables())
		if tmplErr != nil {
			return nil // reported by TestExamples_ParseInTheirDeclaredFormat
		}

		unknown := strictDecode(resolved, ext, target)
		key := filepath.ToSlash(rel)
		known, isKnown := knownSchemaViolations[key]

		switch {
		case len(unknown) > 0 && !isKnown:
			t.Errorf("%s: keys that no %T field accepts: %s\n"+
				"    (production decoding ignores these silently, so the blueprint would be a no-op)",
				path, target, strings.Join(unknown, ", "))
		case len(unknown) > 0 && !strings.Contains(strings.Join(unknown, " "), known.key):
			t.Errorf("%s: expected only the known %q violation, got: %s",
				path, known.key, strings.Join(unknown, ", "))
		case len(unknown) == 0 && isKnown:
			t.Errorf("%s: now decodes cleanly - delete its knownSchemaViolations entry "+
				"so the file cannot regress", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
}

// strictDecode decodes data into target and returns any keys the target has no
// field for. An empty result means the file matches the schema exactly.
func strictDecode(data []byte, format string, target interface{}) []string {
	switch format {
	case "yaml", "yml":
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		if err := dec.Decode(target); err != nil {
			// An empty document (a fully commented-out blueprint) decodes to EOF.
			// That is a legitimate file, not an unknown field.
			if errors.Is(err, io.EOF) {
				return nil
			}
			return unknownFieldsFromError(err.Error())
		}
	case "json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(target); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return unknownFieldsFromError(err.Error())
		}
	case "toml":
		meta, err := toml.Decode(string(data), target)
		if err != nil {
			return unknownFieldsFromError(err.Error())
		}
		var unknown []string
		for _, key := range meta.Undecoded() {
			unknown = append(unknown, key.String())
		}
		return unknown
	}
	return nil
}

// unknownFieldsFromError reduces a decoder error to something readable. Decoders
// phrase this differently, so the message is passed through when it does not
// match a known shape.
func unknownFieldsFromError(msg string) []string {
	var out []string
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// knownTrailingNewlineDrift lists the files where the .toml copy differs from the
// .yaml copy by exactly one thing: the final newline of a multi-line string.
//
// This replaces a blanket "skip everything under alternative_layouts/" exclusion.
// That exclusion was written when the whole subtree had drifted; it no longer
// holds - every other pair under alternative_layouts/ agrees exactly, so the
// exclusion was hiding nothing but suppressing a real check on eleven files.
//
// What remains is narrow and real: YAML's `|` block scalar keeps the trailing
// newline, TOML's `"""` does not unless the source writes one, and these three
// .toml files do not. A .bashrc written from the TOML tree therefore ends without
// a newline where the YAML tree's ends with one. Every other .toml file in
// examples/ writes the `\n`, which is why nothing else is listed here. Fix is in
// examples/, not here; an entry whose file starts agreeing exactly fails the test.
var knownTrailingNewlineDrift = map[string]struct{}{
	"alternative_layouts/flattened/yaml/files.yaml":          {},
	"alternative_layouts/flattened/yaml/scripts.yaml":        {},
	"alternative_layouts/minimal_files/yaml/all_in_one.yaml": {},
}

// trimStringNewlines returns v with the trailing newlines stripped from every
// string it contains, at any depth. Used only to confirm that a mismatch is the
// trailing-newline difference above and nothing else.
func trimStringNewlines(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return strings.TrimRight(t, "\n")
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			out[k] = trimStringNewlines(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = trimStringNewlines(val)
		}
		return out
	default:
		return v
	}
}

// The yaml, json and toml copies of a blueprint must express the same thing.
//
// They had drifted badly - the Arch files example was 194 lines of YAML against
// 31 of JSON, and the macOS scripts example worked in YAML while the JSON and
// TOML copies were missing the `action` every entry needs. Someone reading the
// TOML tree was being shown something that would not run.
func TestExamples_FormatsAgreeWithEachOther(t *testing.T) {
	root := examplesDir(t)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".yaml" {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		parts := strings.Split(rel, string(filepath.Separator))
		idx := -1
		for i, p := range parts {
			if p == "yaml" {
				idx = i
				break
			}
		}
		if idx < 0 {
			return nil
		}

		counterpart := func(format string) string {
			swapped := append([]string{}, parts...)
			swapped[idx] = format
			out := filepath.Join(root, filepath.Join(swapped...))
			return strings.TrimSuffix(out, ".yaml") + "." + format
		}

		jsonPath, tomlPath := counterpart("json"), counterpart("toml")
		if _, err := os.Stat(jsonPath); err != nil {
			return nil
		}
		if _, err := os.Stat(tomlPath); err != nil {
			return nil
		}

		var fromYAML, fromJSON, fromTOML map[string]interface{}
		if !decodeInto(t, path, "yaml", &fromYAML) ||
			!decodeInto(t, jsonPath, "json", &fromJSON) ||
			!decodeInto(t, tomlPath, "toml", &fromTOML) {
			return nil
		}

		// The CUE column is optional per tree (the dead alternative_layouts
		// trees have none) but must agree when present.
		var fromCUE map[string]interface{}
		hasCUE := false
		if cuePath := counterpart("cue"); func() bool { _, statErr := os.Stat(cuePath); return statErr == nil }() {
			hasCUE = decodeInto(t, cuePath, "cue", &fromCUE)
		}

		// An init file names the format its tree is written in, so that one key is
		// required to differ between the copies.
		dropDeclaredFormat(fromYAML)
		dropDeclaredFormat(fromJSON)
		dropDeclaredFormat(fromTOML)
		if hasCUE {
			dropDeclaredFormat(fromCUE)
		}

		key := filepath.ToSlash(rel)
		compare := func(other map[string]interface{}, format string) {
			if reflect.DeepEqual(fromYAML, other) {
				if _, listed := knownTrailingNewlineDrift[key]; listed && format == "toml" {
					t.Errorf("%s and its .toml copy now agree exactly - delete its "+
						"knownTrailingNewlineDrift entry", rel)
				}
				return
			}
			// The only tolerated difference, and only for the files listed below:
			// a block scalar's final newline.
			if _, listed := knownTrailingNewlineDrift[key]; listed && format == "toml" &&
				reflect.DeepEqual(trimStringNewlines(fromYAML), trimStringNewlines(other)) {
				return
			}
			t.Errorf("%s and its .%s copy describe different things", rel, format)
		}
		compare(fromJSON, "json")
		compare(fromTOML, "toml")
		if hasCUE {
			compare(fromCUE, "cue")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking examples: %v", err)
	}
}

func decodeInto(t *testing.T, path, format string, out *map[string]interface{}) bool {
	t.Helper()
	raw, err := os.ReadFile(path) // #nosec G304 -- test walks the repo's own examples tree
	if err != nil {
		t.Errorf("%s: %v", path, err)
		return false
	}
	resolved, err := helpers.ResolveTemplate(raw, exampleVariables())
	if err != nil {
		return false // reported by the parse test
	}
	if err := helpers.UnmarshalBlueprint(resolved, format, out); err != nil {
		return false // reported by the parse test
	}

	// Decoders disagree on Go representation for identical content - yaml yields
	// []interface{} of maps where toml yields []map[string]interface{} - so
	// canonicalise through JSON before comparing, or every file looks different.
	canonical, err := json.Marshal(*out)
	if err != nil {
		t.Errorf("%s: %v", path, err)
		return false
	}
	*out = nil
	if err := json.Unmarshal(canonical, out); err != nil {
		t.Errorf("%s: %v", path, err)
		return false
	}
	return true
}

// dropDeclaredFormat removes blueprints.format, which legitimately differs
// between the yaml, json and toml copies of an init file.
func dropDeclaredFormat(m map[string]interface{}) {
	blueprints, ok := m["blueprints"].(map[string]interface{})
	if !ok {
		return
	}
	delete(blueprints, "format")
}
