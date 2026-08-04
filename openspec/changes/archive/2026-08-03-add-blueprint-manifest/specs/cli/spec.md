# CLI - Deltas

## ADDED Requirements

### Requirement: --config-name selects a manifest entry explicitly

`--config-name <name>` SHALL select the named manifest entry, bypassing
matching and prompting, and SHALL error if the name does not exist. In a
non-TTY run with multiple matches and no `--config-name`, the system SHALL
error listing the candidates rather than prompt.

Why: scripts and CI must never block on a prompt.

#### Scenario: Non-TTY with multiple matches

- **GIVEN** a manifest with two matching entries and stdout not a terminal
- **WHEN** `rwr all` runs without `--config-name`
- **THEN** the run exits nonzero listing `arch-desktop` and `arch-server`

#### Scenario: Explicit selection wins

- **GIVEN** `--config-name arch-server` on a host that would auto-match
  `arch-desktop`
- **WHEN** `rwr all` runs
- **THEN** `arch-server` is used
