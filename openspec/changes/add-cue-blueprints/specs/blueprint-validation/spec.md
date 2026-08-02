# Blueprint Validation — Deltas

## MODIFIED Requirements

### Requirement: CUE errors are validate diagnostics

`rwr validate` SHALL evaluate `.cue` blueprints and report evaluation and
unification failures as diagnostics with file and line, on the same surface
as schema errors for the other formats.

#### Scenario: Constraint violation reported

- **GIVEN** a `.cue` blueprint violating one of its own constraints
- **WHEN** `rwr validate` runs
- **THEN** the diagnostic names the file, line, and failed constraint, and
  validation exits nonzero

### Requirement: Examples cover CUE

`examples/` SHALL cover every blueprint type in CUE as a fourth format column,
validated in CI, per the compatibility contract.

#### Scenario: CI validates CUE examples

- **GIVEN** the examples tree
- **WHEN** CI runs
- **THEN** every `.cue` example validates like its YAML/JSON/TOML siblings
