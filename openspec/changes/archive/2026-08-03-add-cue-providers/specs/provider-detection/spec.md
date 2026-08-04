# Provider Detection - Deltas

## ADDED Requirements

### Requirement: Embedded provider definitions are exported from CUE

Embedded provider definitions SHALL be authored in CUE under `providers/cue/`
and exported at build time to committed JSON under
`internal/system/definitions/providers/`. The binary SHALL embed and decode
the JSON; `cuelang.org/go` SHALL NOT be linked into the binary.

Why: the 25 TOMLs duplicate whole command blocks across families and are
shape-unchecked; CUE unification rejects an invalid provider at export time
instead of at runtime on a user's machine.

#### Scenario: Provider missing a required field fails at build

- **GIVEN** a CUE provider definition lacking `commands.install`
- **WHEN** `mise run providers:export` runs
- **THEN** the export fails naming the provider and field
- **AND** no JSON is produced

#### Scenario: Exported providers decode identically

- **GIVEN** the committed JSON exports
- **WHEN** each is strictly decoded into `types.Provider`
- **THEN** no unknown keys are present and values equal the pre-migration
  TOML-derived values (round-trip test)

### Requirement: Filesystem overrides do not require CUE

Filesystem provider overrides SHALL be accepted as `.toml` or `.json`.
Operators SHALL NOT need a CUE toolchain to override a provider.

#### Scenario: JSON override

- **GIVEN** `~/.config/rwr/providers/pacman.json` copied from the exported JSON
- **WHEN** providers initialize
- **THEN** the override replaces the embedded pacman definition, same as a
  TOML override does today

### Requirement: Committed exports stay fresh

CI SHALL fail when the committed JSON differs from a fresh `cue export` of the
CUE sources, and SHALL run `cue vet` over them.

#### Scenario: Stale export

- **GIVEN** a PR edits `providers/cue/yay.cue` without re-exporting
- **WHEN** CI runs
- **THEN** the `cue-providers` job fails with the diff
