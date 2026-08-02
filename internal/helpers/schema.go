package helpers

import (
	"fmt"

	"github.com/fynxlabs/rwr/internal/types"
)

// DecodeBlueprint reads a blueprint file as the schema version it is written in.
//
// The version is resolved most-specific-first — the file's own `schema_version`,
// then the tree-wide version from the init file, then v1 — and that decides which
// struct the bytes are decoded into. The result is converted to the canonical
// type the processors consume, so nothing past this function needs to know more
// than one version exists.
//
// treeVersion is initConfig.Init.SchemaVersion; pass 0 when unknown.
func DecodeBlueprint(data []byte, format, blueprintType string, treeVersion int) (interface{}, int, error) {
	// Read the declaration first, using the current schema. Every version so far
	// carries schema_version in the same place, which is the one part of the
	// format that cannot move.
	declared, err := declaredSchemaVersion(data, format)
	if err != nil {
		return nil, 0, err
	}

	version := types.ResolveSchemaVersion(declared, treeVersion, blueprintType)
	if err := types.ValidateSchemaVersion(blueprintType, version); err != nil {
		return nil, version, err
	}

	variant, err := types.NewSchemaVariant(blueprintType, version)
	if err != nil {
		return nil, version, err
	}

	if err := UnmarshalBlueprintStrict(data, format, variant.Target()); err != nil {
		return nil, version, fmt.Errorf("reading %s blueprint as schema v%d: %w",
			blueprintType, version, err)
	}

	return variant.Canonical(), version, nil
}

// DecodeBlueprintInto is DecodeBlueprint with the canonical result written into a
// typed target, which is what the processors actually want.
//
// This is the function processors call. DecodeBlueprint returning interface{} is
// why nothing called it: every processor decodes into a concrete struct, and
// nobody was going to write a type assertion at each of twenty call sites. The
// result was that the whole versioning path — resolution, validation, the variant
// registry — sat behind a function with no callers, so a blueprint declaring an
// unsupported schema_version was read as the current schema and applied to the
// machine instead of being refused.
func DecodeBlueprintInto[T any](data []byte, format, blueprintType string, treeVersion int, out *T) error {
	canonical, _, err := DecodeBlueprint(data, format, blueprintType, treeVersion)
	if err != nil {
		return err
	}

	typed, ok := canonical.(*T)
	if !ok {
		return fmt.Errorf("internal: %s schema variant produced %T, expected %T",
			blueprintType, canonical, out)
	}

	*out = *typed
	return nil
}

// TreeSchemaVersion returns the tree-wide schema version an init file declares, or
// 0 when there is no init config to ask.
func TreeSchemaVersion(initConfig *types.InitConfig) int {
	if initConfig == nil {
		return 0
	}
	return initConfig.Init.SchemaVersion
}

// declaredSchemaVersion reads just the schema_version key, ignoring everything
// else, so a file written in a version this build cannot read still reports which
// version it wanted.
func declaredSchemaVersion(data []byte, format string) (int, error) {
	var probe types.SchemaVersion
	if err := UnmarshalBlueprint(data, format, &probe); err != nil {
		// The file is malformed; report that against the real schema rather than
		// this probe, which produces a clearer message.
		return 0, nil //nolint:nilerr // deliberate: surfaced by the real decode below
	}
	return probe.DeclaredVersion(), nil
}
