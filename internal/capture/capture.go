// Package capture turns a handcrafted machine into a blueprint tree: the
// scan layer finds what the operator put there, the selection decides what
// to keep, and generation writes a tree that must validate before capture
// calls itself done.
package capture

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/helpers"
	"github.com/fynxlabs/rwr/internal/scan"
	"github.com/fynxlabs/rwr/internal/types"
	"github.com/fynxlabs/rwr/internal/validate"
)

// Selection is what the operator chose to keep, per category.
type Selection struct {
	Packages map[string][]string // provider → names
	Configs  []scan.ConfigResult
	Services []string
	Git      []scan.GitCheckout
}

// Findings is everything the scan surfaced, for the form to offer.
type Findings struct {
	Packages []scan.PackageResult
	Configs  []scan.ConfigResult
	Services []string
	Git      []scan.GitCheckout
	Home     string
}

// Defaults is the pre-selection: explicit packages and known dotfiles and
// checkouts in, services out (most are distro plumbing), secret-shaped
// paths never.
func Defaults(findings Findings) Selection {
	selection := Selection{Packages: map[string][]string{}}
	for _, result := range findings.Packages {
		if !result.Unfiltered {
			selection.Packages[result.Provider] = result.Names
		}
	}
	for _, config := range findings.Configs {
		if config.Known && !scan.SecretShaped(config.Rel) {
			selection.Configs = append(selection.Configs, config)
		}
	}
	selection.Git = findings.Git
	return selection
}

// Generate writes the tree: init, per-category blueprints, and the selected
// config files copied under files/src/. The generated tree is validated
// before success is reported — a generator whose output needs hand-repair
// has not captured anything.
func Generate(out io.Writer, dir, format string, selection Selection, findings Findings, manifest bool, osInfo *types.OSInfo) error {
	ext := helpers.BlueprintExtension(format)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	initDoc := map[string]interface{}{
		"blueprints": map[string]interface{}{
			"format":   format,
			"location": ".",
		},
	}
	if err := writeDoc(filepath.Join(dir, "init"+ext), initDoc, format); err != nil {
		return err
	}

	if len(selection.Packages) > 0 {
		var results []scan.PackageResult
		for provider, names := range selection.Packages {
			if len(names) > 0 {
				results = append(results, scan.PackageResult{Provider: provider, Names: names})
			}
		}
		if len(results) > 0 {
			block, err := scan.EmitPackages(results, format)
			if err != nil {
				return err
			}
			if err := writeFile(filepath.Join(dir, "packages", "packages"+ext), block); err != nil {
				return err
			}
		}
	}

	if len(selection.Services) > 0 {
		block, err := scan.EmitServices(selection.Services, format)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "services", "services"+ext), block); err != nil {
			return err
		}
	}

	if len(selection.Git) > 0 {
		block, err := scan.EmitGit(selection.Git, findings.Home, format)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "git", "git"+ext), block); err != nil {
			return err
		}
	}

	if len(selection.Configs) > 0 {
		for _, config := range selection.Configs {
			if scan.SecretShaped(config.Rel) {
				helpers.Say(out, "warning: capturing secret-shaped path %s — review before committing\n", config.Rel)
			}
			dest := filepath.Join(dir, "files", "src", config.Rel)
			if err := copyPath(config.Path, dest); err != nil {
				return fmt.Errorf("copying %s: %w", config.Rel, err)
			}
		}
		block, err := scan.EmitConfigs(selection.Configs, findings.Home, format)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "files", "files"+ext), block); err != nil {
			return err
		}
	}

	if manifest {
		entry := map[string]interface{}{
			"name": "captured",
			"init": "init" + ext,
		}
		if osInfo != nil {
			if osInfo.System.OS != "" {
				entry["os"] = osInfo.System.OS
			}
			if osInfo.System.OSFamily != "" && osInfo.System.OSFamily != osInfo.System.OS {
				entry["distro"] = osInfo.System.OSFamily
			}
		}
		doc := map[string]interface{}{"configurations": []interface{}{entry}}
		if err := writeDoc(filepath.Join(dir, "manifest"+ext), doc, format); err != nil {
			return err
		}
	}

	results, err := validate.Validate(types.ValidationOptions{Path: dir, ValidateBlueprints: true}, osInfo)
	if err != nil {
		return fmt.Errorf("validating the generated tree: %w", err)
	}
	if results.ErrorCount > 0 {
		for _, issue := range results.Issues {
			if issue.Severity == types.ValidationError {
				helpers.Say(out, "  ✗ %s: %s\n", issue.File, issue.Message)
			}
		}
		return fmt.Errorf("the generated tree fails validation with %d error(s) — capture aborted, fix the selection and re-run", results.ErrorCount)
	}
	helpers.Say(out, "Captured to %s (validated).\n", dir)
	return nil
}

func writeDoc(path string, doc map[string]interface{}, format string) error {
	encoded, err := helpers.EncodeBlueprintDoc(doc, format)
	if err != nil {
		return err
	}
	return writeFile(path, encoded)
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644) // #nosec G306 -- generated blueprint tree, world-readable config
}

// copyPath copies a file or directory tree.
func copyPath(source, dest string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(source) // #nosec G304 -- operator-selected config from their own home
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, info.Mode().Perm()) // #nosec G306 -- preserves the source's own mode
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyPath(filepath.Join(source, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
