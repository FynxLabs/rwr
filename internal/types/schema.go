package types

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultSchemaVersion is the version assumed for a blueprint type that has only
// ever had one. It is not the fallback for an undeclared blueprint — see
// ResolveSchemaVersion, which falls back to the latest version instead.
const DefaultSchemaVersion = 1

// supportedSchemaVersions lists the versions each blueprint type can be written
// in. Versions are per type on purpose: a breaking change to packages moves
// packages to v2 and leaves everything else at v1, instead of dragging the whole
// schema forward and arriving at v11 over changes that each touched one resource.
//
// To introduce a v2 for a type: add 2 here, teach that type's processor to branch
// on the resolved version, and document the difference. Nothing else moves.
var supportedSchemaVersions = map[string][]int{
	BlueprintTypePackages:        {1},
	BlueprintTypeRepositories:    {1},
	BlueprintTypeFiles:           {1},
	BlueprintTypeServices:        {1},
	BlueprintTypeGit:             {1},
	BlueprintTypeScripts:         {1},
	BlueprintTypeSSHKeys:         {1},
	BlueprintTypeFonts:           {1},
	BlueprintTypeUsers:           {1},
	BlueprintTypeConfiguration:   {1},
	BlueprintTypePackageManagers: {1},
	BlueprintTypeBootstrap:       {1},
}

// SchemaVersion is embedded in every blueprint data struct so an individual file
// can declare the schema it is written in:
//
//	schema_version: 2
//	packages:
//	  - name: git
//
// A file's declaration overrides the tree-wide version from the init file, which
// is what makes a single-resource breaking change possible: move one packages
// blueprint to v2 and leave the rest of the tree alone.
type SchemaVersion struct {
	SchemaVersion int `mapstructure:"schema_version,omitempty" yaml:"schema_version,omitempty" json:"schema_version,omitempty" toml:"schema_version,omitempty"`
}

// DeclaredVersion returns the version this file declares, or 0 if it declares
// none.
func (s SchemaVersion) DeclaredVersion() int { return s.SchemaVersion }

// ResolveSchemaVersion decides which schema version a blueprint file should be
// read as.
//
// Precedence is most-specific-wins: the file's own declaration, then the
// tree-wide version from the init file, then the latest version this build
// supports for that type.
//
// Falling back to the latest means a blueprint written today gets today's schema
// without boilerplate, and the version field is what you add when you want to be
// pinned rather than something you must add to get started. The consequence is
// that an undeclared blueprint follows the schema forward across upgrades, so a
// tree that needs to stay on a particular version has to say so — which is what
// the declaration is for.
func ResolveSchemaVersion(fileDeclared, treeDeclared int, blueprintType string) int {
	if fileDeclared > 0 {
		return fileDeclared
	}
	if treeDeclared > 0 {
		return treeDeclared
	}
	return LatestSchemaVersion(blueprintType)
}

// LatestSchemaVersion returns the newest version this build can read for a
// blueprint type, or DefaultSchemaVersion for a type it does not know.
func LatestSchemaVersion(blueprintType string) int {
	latest := 0
	for _, v := range supportedSchemaVersions[blueprintType] {
		if v > latest {
			latest = v
		}
	}
	if latest == 0 {
		return DefaultSchemaVersion
	}
	return latest
}

// ValidateSchemaVersion reports whether this build can read blueprintType at the
// given version.
//
// It is deliberately strict. A blueprint asking for a schema this binary does not
// know is one whose fields would be misread, and misreading a blueprint means
// doing the wrong thing to somebody's machine rather than refusing.
func ValidateSchemaVersion(blueprintType string, version int) error {
	versions, known := supportedSchemaVersions[blueprintType]
	if !known {
		return fmt.Errorf("unknown blueprint type %q", blueprintType)
	}
	if version <= 0 {
		return fmt.Errorf("%s: %d is not a valid schema version", blueprintType, version)
	}
	for _, v := range versions {
		if v == version {
			return nil
		}
	}
	return fmt.Errorf("%s: schema version %d is not supported by this build (supports %s) — "+
		"upgrade rwr, or write this blueprint in a supported version",
		blueprintType, version, versionList(blueprintType))
}

// ValidateTreeSchemaVersion reports whether a tree-wide version from an init file
// is readable for every blueprint type. A tree-wide declaration applies to
// everything, so it has to be supported everywhere.
func ValidateTreeSchemaVersion(version int) error {
	if version == 0 {
		return nil // undeclared, resolves to the default
	}
	if version < 0 {
		return fmt.Errorf("%d is not a valid schema version", version)
	}

	var unsupported []string
	for _, blueprintType := range sortedKeys(supportedSchemaVersions) {
		if !supportsVersion(blueprintType, version) {
			unsupported = append(unsupported, fmt.Sprintf("%s (supports %s)",
				blueprintType, versionList(blueprintType)))
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("schema version %d is not supported for %s — "+
		"declare it per blueprint file instead, or upgrade rwr",
		version, strings.Join(unsupported, ", "))
}

func supportsVersion(blueprintType string, version int) bool {
	for _, v := range supportedSchemaVersions[blueprintType] {
		if v == version {
			return true
		}
	}
	return false
}

// versionList renders the versions a blueprint type supports, for error messages.
func versionList(blueprintType string) string {
	versions := supportedSchemaVersions[blueprintType]
	parts := make([]string, 0, len(versions))
	for _, v := range versions {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, ", ")
}

// SupportedSchemaVersions returns the versions a blueprint type can be written in.
func SupportedSchemaVersions(blueprintType string) []int {
	out := append([]int(nil), supportedSchemaVersions[blueprintType]...)
	sort.Ints(out)
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
