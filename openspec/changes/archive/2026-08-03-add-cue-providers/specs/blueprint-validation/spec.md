# Blueprint Validation - Deltas

## ADDED Requirements

### Requirement: Embedded provider contracts live in the CUE schema

Embedded provider contracts SHALL be expressed in the CUE schema - the checks
currently hand-rolled in `internal/validate/providers.go`: required `name`,
`detection.binary`, `commands.install`; step `action` constrained to the enum
the processors implement; `condition` restricted to derivable predicate names;
no literal `/tmp/` paths in steps. Go validation SHALL remain only for
filesystem overrides.

Why: the hand-rolled Go checks have already drifted from the processors once
(fictional actions, stale fields); a schema the export gate enforces cannot
drift silently.

#### Scenario: Invalid action rejected at export

- **GIVEN** a CUE provider step with `action: "instal"`
- **WHEN** the export runs
- **THEN** it fails listing the allowed actions

#### Scenario: Predicate list cannot drift

- **GIVEN** the CUE `or` list of condition predicates and Go's
  `repositoryPredicates` keys
- **WHEN** the cross-check test runs
- **THEN** it fails if the two sets differ in either direction
