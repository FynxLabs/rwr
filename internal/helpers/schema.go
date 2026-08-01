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

	version := types.ResolveSchemaVersion(declared, treeVersion)
	if err := types.ValidateSchemaVersion(blueprintType, version); err != nil {
		return nil, version, err
	}

	variant, err := types.NewSchemaVariant(blueprintType, version)
	if err != nil {
		return nil, version, err
	}

	if err := UnmarshalBlueprint(data, format, variant.Target()); err != nil {
		return nil, version, fmt.Errorf("reading %s blueprint as schema v%d: %w",
			blueprintType, version, err)
	}

	return variant.Canonical(), version, nil
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
