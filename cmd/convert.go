package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newConvertCmd converts a blueprint tree between formats and migrates
// deprecated constructs to their current equivalents. Dry-run by default:
// it prints what would change; --write applies.
func newConvertCmd() *cobra.Command {
	var (
		toFormat string
		migrate  bool
		write    bool
	)

	convertCmd := &cobra.Command{
		Use:   "convert [path]",
		Short: "Convert a blueprint tree between formats, or migrate deprecated constructs",
		Long: `Convert every blueprint, init, bootstrap, and manifest file in a tree to
another format (--to yaml|json|toml|cue), or rewrite deprecated constructs to
their current equivalents (--migrate).

Dry-run by default: nothing is written without --write. Comments are NOT
preserved across formats — the command warns per file that carries them.
Template placeholders survive as quoted strings; a file whose templates make
it unparseable is reported and skipped, never mangled.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			if toFormat == "" && !migrate {
				return fmt.Errorf("nothing to do: pass --to <format> and/or --migrate")
			}
			if toFormat != "" {
				canonical, err := helpers.CanonicalFormat(toFormat)
				if err != nil {
					return err
				}
				toFormat = canonical
			}
			return runConvert(cmd, root, toFormat, migrate, write)
		},
	}

	convertCmd.Flags().StringVar(&toFormat, "to", "", "Target format: yaml, json, toml, or cue")
	convertCmd.Flags().BoolVar(&migrate, "migrate", false, "Rewrite deprecated constructs (init-file inline sections) to their current form")
	convertCmd.Flags().BoolVar(&write, "write", false, "Apply the changes (default is a dry run)")
	return convertCmd
}

// say writes progress lines; a broken stdout must not abort a conversion
// midway through a tree.
func say(w interface{ Write([]byte) (int, error) }, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck
}

func runConvert(cmd *cobra.Command, root, toFormat string, migrate, write bool) error {
	out := cmd.OutOrStdout()
	changed := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != root && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !helpers.IsBlueprintFile(path) {
			return nil
		}

		format, err := helpers.FormatForPath(path)
		if err != nil {
			return nil
		}

		data, err := os.ReadFile(path) // #nosec G304 G122 -- operator's own tree, walked read-only
		if err != nil {
			say(out, "  ! %s: %v\n", path, err)
			return nil
		}

		var doc map[string]interface{}
		if err := helpers.UnmarshalBlueprint(data, format, &doc); err != nil {
			say(out, "  ! %s: cannot parse (%v) — skipped, not mangled\n", path, err)
			return nil
		}

		if hasComments(data, format) {
			say(out, "  ~ %s carries comments; they are NOT preserved across formats\n", path)
		}

		if migrate {
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			if base == "init" {
				n, migErr := migrateInitInlineSections(out, path, doc, format, write)
				if migErr != nil {
					return migErr
				}
				changed += n
			}
		}

		if toFormat != "" && toFormat != format {
			target := strings.TrimSuffix(path, filepath.Ext(path)) + extensionFor(toFormat)
			encoded, encErr := encodeAs(doc, toFormat)
			if encErr != nil {
				say(out, "  ! %s: cannot encode as %s: %v\n", path, toFormat, encErr)
				return nil
			}
			if write {
				if wErr := os.WriteFile(target, encoded, 0o644); wErr != nil { // #nosec G306 G122 -- blueprint files are world-readable config; operator's own tree
					return wErr
				}
				if rmErr := os.Remove(path); rmErr != nil { // #nosec G122 -- removing the file just converted, operator's own tree
					return rmErr
				}
			}
			say(out, "  → %s => %s\n", path, target)
			changed++
		}
		return nil
	})
	if err != nil {
		return err
	}

	if changed == 0 {
		say(out, "nothing to change"+"\n")
	} else if !write {
		say(out, "%d change(s); re-run with --write to apply\n", changed)
	}
	return nil
}

// migrateInitInlineSections moves the removed inline resource sections out of
// an init file into blueprint files — the first migrateRules entry.
func migrateInitInlineSections(out interface{ Write([]byte) (int, error) }, initPath string, doc map[string]interface{}, format string, write bool) (int, error) {
	sections := map[string]string{
		"repositories": "repositories", "packages": "packages", "services": "services",
		"files": "files", "templates": "files", "directories": "files", "configuration": "configuration",
	}
	root := filepath.Dir(initPath)
	moved := 0
	for key, dir := range sections {
		value, present := doc[key]
		if !present {
			continue
		}
		targetDir := filepath.Join(root, dir)
		target := filepath.Join(targetDir, "from-init"+extensionFor(format))
		say(out, "  → %s: move %q into %s\n", initPath, key, target)
		if write {
			if err := os.MkdirAll(targetDir, 0o755); err != nil { // #nosec G301 -- operator's own blueprint tree
				return moved, err
			}
			payload := map[string]interface{}{key: value}
			if existing, err := os.ReadFile(target); err == nil { // #nosec G304 -- operator's own tree
				var current map[string]interface{}
				if helpers.UnmarshalBlueprint(existing, format, &current) == nil {
					current[key] = value
					payload = current
				}
			}
			encoded, err := encodeAs(payload, format)
			if err != nil {
				return moved, err
			}
			if err := os.WriteFile(target, encoded, 0o644); err != nil { // #nosec G306 -- blueprint files are world-readable config
				return moved, err
			}
			delete(doc, key)
			rewritten, err := encodeAs(doc, format)
			if err != nil {
				return moved, err
			}
			if err := os.WriteFile(initPath, rewritten, 0o644); err != nil { // #nosec G306
				return moved, err
			}
		}
		moved++
	}
	return moved, nil
}

// encodeAs renders a document in a target format. CUE output is JSON-form
// CUE: valid, lossless, mechanical — idiomatic CUE is authoring work.
func encodeAs(doc map[string]interface{}, format string) ([]byte, error) {
	switch format {
	case "yaml":
		return yaml.Marshal(doc)
	case "json":
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "toml":
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "cue":
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "\t")
		if err := enc.Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("unsupported target format %q", format)
}

func extensionFor(format string) string {
	return "." + format
}

// hasComments detects comment markers well enough to warn (never to block).
func hasComments(data []byte, format string) bool {
	marker := "#"
	if format == "json" {
		return false
	}
	if format == "cue" {
		marker = "//"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			return true
		}
	}
	return false
}
