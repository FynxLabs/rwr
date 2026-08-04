# Initialization - Deltas

## ADDED Requirements

### Requirement: Init file discovery uses the format registry

Init and bootstrap file discovery SHALL derive candidate filenames
(`init.<ext>`, `bootstrap.<ext>`) from the format registry rather than
hardcoded lists.

#### Scenario: New format is discoverable without touching discovery code

- **GIVEN** a format is added to the registry
- **WHEN** a directory contains `init.<newext>`
- **THEN** init discovery finds it with no change to `cmd/` or `processors/`

### Requirement: --init-file flag and config key agree

The `--init-file` flag SHALL be bound to the same configuration key it is read
from. Setting the value via config file SHALL behave identically to the flag.

Why: today the flag binds to `rwr.init-file` but resolution reads
`repository.init-file`, so the documented config key silently ignores the flag
binding.

#### Scenario: Config key resolves like the flag

- **GIVEN** a config file setting the init-file key to `/tmp/x/init.yaml`
- **WHEN** `rwr all` runs without `--init-file`
- **THEN** `/tmp/x/init.yaml` is used, same as if passed via `--init-file`
