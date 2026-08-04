package scan

import (
	"path"

	"github.com/fynxlabs/rwr/internal/helpers"
)

// EmitPackages renders package results as a packages blueprint block in the
// given registry format. The block strict-decodes against the blueprint
// schema - emission that produces invalid blueprints is worse than a list.
func EmitPackages(results []PackageResult, format string) ([]byte, error) {
	var entries []interface{}
	for _, result := range results {
		entries = append(entries, map[string]interface{}{
			"names":           result.Names,
			"action":          "install",
			"package_manager": result.Provider,
		})
	}
	return helpers.EncodeBlueprintDoc(map[string]interface{}{"packages": entries}, format)
}

// EmitGit renders discovered checkouts as a git blueprint block; paths under
// home render portable via the User.home template.
func EmitGit(checkouts []GitCheckout, home, format string) ([]byte, error) {
	var entries []interface{}
	for _, checkout := range checkouts {
		if checkout.URL == "" {
			continue
		}
		entries = append(entries, map[string]interface{}{
			"name":   path.Base(checkout.Path),
			"action": "clone",
			"url":    checkout.URL,
			"path":   templatedHome(checkout.Path, home),
		})
	}
	return helpers.EncodeBlueprintDoc(map[string]interface{}{"git": entries}, format)
}

// EmitServices renders enabled units as a services blueprint block.
func EmitServices(services []string, format string) ([]byte, error) {
	var entries []interface{}
	for _, service := range services {
		entries = append(entries, map[string]interface{}{
			"name":   service,
			"action": "enable",
		})
	}
	return helpers.EncodeBlueprintDoc(map[string]interface{}{"services": entries}, format)
}

// EmitConfigs renders selected configs as file entries whose source is the
// captured copy under files/src/ and whose target is the origin, portable
// via the User.home template.
func EmitConfigs(configs []ConfigResult, home, format string) ([]byte, error) {
	var entries []interface{}
	for _, config := range configs {
		entries = append(entries, map[string]interface{}{
			"name":   path.Base(config.Rel),
			"action": "copy",
			"source": "src/" + config.Rel,
			"target": templatedHome(path.Dir(config.Path), home) + "/",
		})
	}
	return helpers.EncodeBlueprintDoc(map[string]interface{}{"files": entries}, format)
}

// templatedHome rewrites an absolute path under home as a User.home
// template, so the emitted blueprint works on a machine with another
// username.
func templatedHome(p, home string) string {
	if home != "" && len(p) >= len(home) && p[:len(home)] == home {
		return "{{ .User.home }}" + p[len(home):]
	}
	return p
}
