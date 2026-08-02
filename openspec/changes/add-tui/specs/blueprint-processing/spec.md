# Blueprint Processing — Deltas

## ADDED Requirements

### Requirement: Two-stage resolve produces a Plan before execution

The system SHALL resolve blueprints into a `Plan` before executing: stage 1
(parse, imports, schema, templates, diagnostics) requires only paths + OS
detection; stage 2 (provider states, resource enumeration) SHALL run only
after init and bootstrap complete.

Why: bootstrap can install the package manager later blueprints depend on;
detecting providers earlier produces wrong results.

#### Scenario: validate uses stage 1 only

- **GIVEN** `rwr validate`
- **WHEN** it runs
- **THEN** it consumes stage 1 output and performs no provider detection

## MODIFIED Requirements

### Requirement: Processor errors accumulate in non-interactive runs

In non-interactive mode, an error from a processor SHALL be recorded and the
run SHALL continue with the remaining processors; the exit code SHALL be
nonzero if any error was recorded. Pre-loop failures (blueprint fetch, missing
location, package-manager install, unresolvable run order) SHALL abort in both
modes. Interactive mode SHALL halt at the failing processor.

Why: a provisioning run that dies at processor 2 of 13 leaves the machine in a
worse state than one that applies everything it can and reports what failed.
Logging a failure and exiting 0 is a bug (repo invariant).

#### Scenario: Non-interactive continues past a failure

- **GIVEN** `--interactive=false` and a failing `packages` processor
- **WHEN** `rwr all` runs
- **THEN** the remaining processors execute
- **AND** the exit code is nonzero and the failure appears in the summary
