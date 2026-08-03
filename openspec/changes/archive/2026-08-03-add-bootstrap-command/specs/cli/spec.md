# CLI — Deltas

## ADDED Requirements

### Requirement: Bootstrap runs standalone

RWR SHALL provide `rwr run bootstrap` (and the root shorthand `rwr bootstrap`)
to run the bootstrap processor by itself. An explicit invocation SHALL run the
bootstrap even when the run-once marker exists, SHALL refresh the marker on
success, and SHALL fail with an error naming the candidate filenames when the
tree has no bootstrap file. The gating of bootstrap inside `rwr all` — skipped
when the marker exists unless `--force-bootstrap` is given — is unchanged.

#### Scenario: Re-running bootstrap after editing it

- **WHEN** the run-once marker exists and the operator runs `rwr bootstrap`
- **THEN** the bootstrap processor runs, and no other processor does

#### Scenario: No bootstrap file in the tree

- **WHEN** `rwr bootstrap` runs against a tree with no bootstrap file in any
  supported format
- **THEN** the command exits non-zero with an error naming the filenames it
  looked for
