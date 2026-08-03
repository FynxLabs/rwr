# CLI — Deltas

## ADDED Requirements

### Requirement: Config is viewable and editable, with secrets redacted

RWR SHALL provide `rwr config view` showing the effective merged
configuration with credential values redacted unless `--show-secrets` is
given, `rwr config edit` opening the config file in the operator's editor
(creating a default config first when none exists, and warning when the
edited file no longer parses), and `rwr config create` as the subcommand form
of `--create`, which SHALL remain as a deprecated alias.

#### Scenario: Viewing a config that holds a token

- **WHEN** the config file contains `repository.gh_api_token` and the
  operator runs `rwr config view`
- **THEN** the token's value appears as `[redacted]`
- **AND** appears in clear only with `--show-secrets`

#### Scenario: Editing with no config file

- **WHEN** `rwr config edit` runs and no config file exists
- **THEN** a default config is created and the editor opens it

### Requirement: Common flags have single-letter shorts

RWR SHALL accept `-n` as the short form of `--dry-run` and `-l` as the short
form of `--log-level`. Existing shorts (`-d` debug, `-i` init-file, `-I`
interactive, `-p` profile) SHALL keep their current meanings.

#### Scenario: Short-form dry run

- **WHEN** the operator runs `rwr all -n`
- **THEN** the run behaves exactly as with `--dry-run`
