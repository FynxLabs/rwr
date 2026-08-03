# state-tracking Specification

## Purpose
TBD - created by archiving change add-uninstall-status. Update Purpose after archive.
## Requirements
### Requirement: Applies are journaled to a run record

RWR SHALL record every resource it applies during a non-dry-run to a versioned,
append-only run record under the configuration directory, with per-resource
identity sufficient to check or reverse the apply (package name and provider,
file destination and content hash, service name and action, repository name,
font name, git target). The record SHALL be written incrementally so an
interrupted run still leaves a truthful partial record, and SHALL contain no
secret values.

#### Scenario: A run journals what it applied

- **WHEN** `rwr all` installs a package and writes a file
- **THEN** the run record contains an entry for each, identifying the package
  by name and provider and the file by destination and content hash

#### Scenario: Dry-run leaves no record

- **WHEN** `rwr all --dry-run` completes
- **THEN** no run record is written

### Requirement: Uninstall reverses only what is honestly reversible

RWR SHALL drive `rwr uninstall` from the run record, in reverse application
order, and SHALL enumerate the recorded actions it cannot reverse (scripts,
configuration writes, users and groups, uploaded SSH keys) before making any
change. RWR SHALL delete a recorded file only when its content hash still
matches the record, and SHALL refuse to uninstall when no run record exists.

#### Scenario: Modified file is not deleted

- **WHEN** a recorded file's content no longer matches its recorded hash
- **THEN** `rwr uninstall` skips it and lists it as modified

#### Scenario: No record, no guessing

- **WHEN** `rwr uninstall` runs on a machine with no run record
- **THEN** it exits with an error explaining that nothing was recorded, and
  changes nothing

