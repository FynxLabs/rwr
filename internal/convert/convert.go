// Package convert rewrites blueprint trees: between formats (--to) and from
// deprecated constructs to their current equivalents (--migrate). Dry-run by
// default; nothing is written unless write is set.
package convert

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fynxlabs/rwr/internal/helpers"
)

// Run walks root and converts every blueprint, init, bootstrap, and manifest
// file: to toFormat when set (and different from the file's own), and through
// the migration rules when migrate is set. Comments are not preserved across
// formats - each file carrying them is warned about. A file whose template
// placeholders make it unparseable is reported and skipped, never mangled.
func Run(out io.Writer, root, toFormat string, migrate, write bool) error {
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
			helpers.Say(out, "  ! %s: %v\n", path, err)
			return nil
		}

		var doc map[string]interface{}
		if err := helpers.UnmarshalBlueprint(data, format, &doc); err != nil {
			helpers.Say(out, "  ! %s: cannot parse (%v) - skipped, not mangled\n", path, err)
			return nil
		}

		if hasComments(data, format) {
			helpers.Say(out, "  ~ %s carries comments; they are NOT preserved across formats\n", path)
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
			target := strings.TrimSuffix(path, filepath.Ext(path)) + helpers.BlueprintExtension(toFormat)
			encoded, encErr := helpers.EncodeBlueprintDoc(doc, toFormat)
			if encErr != nil {
				helpers.Say(out, "  ! %s: cannot encode as %s: %v\n", path, toFormat, encErr)
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
			helpers.Say(out, "  → %s => %s\n", path, target)
			changed++
		}
		return nil
	})
	if err != nil {
		return err
	}

	if changed == 0 {
		helpers.Say(out, "nothing to change\n")
	} else if !write {
		helpers.Say(out, "%d change(s); re-run with --write to apply\n", changed)
	}
	return nil
}

// migrateInitInlineSections moves the removed inline resource sections out of
// an init file into blueprint files - the first migration rule.
func migrateInitInlineSections(out io.Writer, initPath string, doc map[string]interface{}, format string, write bool) (int, error) {
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
		target := filepath.Join(targetDir, "from-init"+helpers.BlueprintExtension(format))
		helpers.Say(out, "  → %s: move %q into %s\n", initPath, key, target)
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
			encoded, err := helpers.EncodeBlueprintDoc(payload, format)
			if err != nil {
				return moved, err
			}
			if err := os.WriteFile(target, encoded, 0o644); err != nil { // #nosec G306 -- blueprint files are world-readable config
				return moved, err
			}
			delete(doc, key)
			rewritten, err := helpers.EncodeBlueprintDoc(doc, format)
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
