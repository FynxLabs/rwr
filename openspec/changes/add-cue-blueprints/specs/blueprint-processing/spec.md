# Blueprint Processing — Deltas

## MODIFIED Requirements

### Requirement: CUE is a supported blueprint format

`.cue` SHALL be a registered blueprint format alongside YAML, JSON, and TOML,
valid for blueprints, imports, init files, bootstrap files, and the manifest.
A `.cue` file SHALL be evaluated in-process (`cuelang.org/go`), exported to
concrete values, and decoded through the existing strict path — semantics
identical to the equivalent YAML.

Why: CUE gives authors types, constraints, and composition at authoring time;
errors surface before a run touches the machine. Evaluation is in-process
because rwr bootstraps machines that do not have a `cue` binary.

#### Scenario: CUE blueprint equals its YAML twin

- **GIVEN** `packages.cue` and `packages.yaml` declaring the same packages
- **WHEN** both are decoded
- **THEN** the resulting blueprint values are identical

#### Scenario: Non-concrete value is an error

- **GIVEN** a `.cue` blueprint with an unresolved field (`version: string`)
- **WHEN** it is evaluated
- **THEN** decoding fails with the field name and file/line

### Requirement: CUE evaluation is sandboxed to the blueprint tree

CUE evaluation SHALL NOT resolve modules, imports, or embeds from the
network or from filesystem paths outside the blueprint tree.

Why: blueprints are untrusted input; evaluation must not become a way to read
arbitrary files or phone home.

#### Scenario: Escape rejected

- **GIVEN** a `.cue` file importing a path outside the blueprint tree
- **WHEN** it is evaluated
- **THEN** evaluation fails with a containment error
