# System Scan - Deltas

## ADDED Requirements

### Requirement: Scans report what the operator put on the machine

A scan SHALL report the machine's operator-chosen state per category -
packages, configs, services, git checkouts - preferring each package
manager's explicitly-installed query (`list_explicit`) over its full list,
and marking results unfiltered when only the full list exists. Dependency
noise is the difference between a usable answer and 1200 rows.

#### Scenario: Explicit packages preferred

- **GIVEN** a provider defining both `list` and `list_explicit`
- **WHEN** the package scanner runs
- **THEN** the result contains the explicit set and is not marked unfiltered

#### Scenario: Fallback is honest

- **WHEN** a provider defines only `list`
- **THEN** the scan result carries the full set marked unfiltered

### Requirement: Scans are read-only and unelevated

A scan SHALL NOT mutate the system and SHALL NOT elevate: it executes only
the provider's list verbs and reads files and unit states. A state
inspection that changes state, or asks for root, is one nobody trusts.

#### Scenario: Only list verbs execute

- **GIVEN** a provider whose non-list commands are trap binaries
- **WHEN** a scan runs
- **THEN** no trap fires

### Requirement: Config scanning ships a noise exclusion list

The config scanner SHALL report the known dotfiles and top-level `~/.config`
entries, minus a shipped exclusion list of cache, state, and session
directories; excluded entries SHALL be recoverable with an
include-everything flag. The human selects from what is shown, so what is
shown must be worth reading.

#### Scenario: Cache noise excluded by default

- **GIVEN** `~/.config` containing an application config and `~/.config/pulse`
- **WHEN** the config scanner runs
- **THEN** the application config is reported and the pulse state is not

### Requirement: Scan results render as blueprints in any format

A scan result SHALL render as a blueprint block in any registry format
(cue, yaml, json, toml), and the rendered block SHALL strict-decode against
the blueprint schema. Emission that produces invalid blueprints is worse
than a plain list.

#### Scenario: Rendered block round-trips

- **WHEN** a package scan renders as a cue packages block
- **THEN** the block decodes strictly as a packages blueprint with the same
  entries
