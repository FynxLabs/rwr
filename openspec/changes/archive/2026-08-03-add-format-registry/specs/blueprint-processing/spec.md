# Blueprint Processing — Deltas

## ADDED Requirements

### Requirement: Format registry is the single source of blueprint format knowledge

The system SHALL resolve a blueprint file's format from its extension through a
single registry. No other code SHALL enumerate blueprint extensions or map an
extension to a decoder.

Why: format knowledge is duplicated across ~15 sites today; each new format is
N hand edits, and the copies have already diverged (see next requirement).

#### Scenario: Unknown extension yields a diagnostic, not a panic

- **GIVEN** a blueprint directory containing `packages.xml`
- **WHEN** the run order is processed
- **THEN** the file is reported as an unsupported format with its path
- **AND** the run does not panic (today `all.go:148` panics on an extensionless file)

### Requirement: Format is derived per file

The system SHALL determine each blueprint file's format from that file's
extension. `Init.Format` SHALL NOT cause a file with a recognized extension to
be decoded as another format.

Why: half the codebase assumes a tree-uniform format via `Init.Format`, half
derives per file; a mixed tree behaves differently depending on which code path
touches it.

#### Scenario: Mixed-format tree

- **GIVEN** a blueprint tree containing `packages.yaml` and `files.toml`
- **WHEN** `rwr all` processes the tree
- **THEN** each file is decoded according to its own extension
- **AND** profile discovery and file ordering see both files
