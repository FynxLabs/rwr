package validate

import (
	"bytes"
	"encoding/json"
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

// Every example must decode into its blueprint struct with no unknown fields.
//
// This is the backwards-compatibility check: the decoders RWR uses in production
// ignore unknown keys, so renaming or removing a struct field silently turns
// every blueprint using it into a no-op rather than an error. Decoding strictly
// here means such a change fails the build with the field named, instead of
// shipping and quietly doing nothing on users' machines.
//
// It also catches the reverse — an example inventing a field that never existed,
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

		// The directory name is how RWR itself decides the blueprint type
		// (getProcessorType in processors/blueprints.go).
		kind := filepath.Base(filepath.Dir(path))
		target := blueprintTarget(kind)
		if target == nil {
			return nil // init files, bootstrap, and the layout examples
		}

		raw, readErr := os.ReadFile(path) // #nosec G304 -- test walks the repo's own examples tree
		if readErr != nil {
			t.Errorf("%s: %v", path, readErr)
			return nil
		}
		resolved, tmplErr := helpers.ResolveTemplate(raw, exampleVariables())
		if tmplErr != nil {
			return nil // reported by TestExamples_ParseInTheirDeclaredFormat
		}

		if unknown := strictDecode(resolved, ext, target); len(unknown) > 0 {
			t.Errorf("%s: keys that no %T field accepts: %s\n"+
				"    (production decoding ignores these silently, so the blueprint would be a no-op)",
				path, target, strings.Join(unknown, ", "))
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
			return unknownFieldsFromError(err.Error())
		}
	case "json":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(target); err != nil {
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

// The yaml, json and toml copies of a blueprint must express the same thing.
//
// They had drifted badly — the Arch files example was 194 lines of YAML against
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
		// alternative_layouts/ demonstrates a layout the code does not implement
		// (type detection by file content); it is being reworked separately, and
		// its copies have drifted along with everything else there.
		if strings.HasPrefix(rel, "alternative_layouts"+string(filepath.Separator)) {
			return nil
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

		// An init file names the format its tree is written in, so that one key is
		// required to differ between the copies.
		dropDeclaredFormat(fromYAML)
		dropDeclaredFormat(fromJSON)
		dropDeclaredFormat(fromTOML)

		if !reflect.DeepEqual(fromYAML, fromJSON) {
			t.Errorf("%s and its .json copy describe different things", rel)
		}
		if !reflect.DeepEqual(fromYAML, fromTOML) {
			t.Errorf("%s and its .toml copy describe different things", rel)
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

	// Decoders disagree on Go representation for identical content — yaml yields
	// []interface{} of maps where toml yields []map[string]interface{} — so
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
