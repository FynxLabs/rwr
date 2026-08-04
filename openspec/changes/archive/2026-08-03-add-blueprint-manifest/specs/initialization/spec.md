# Initialization - Deltas

## ADDED Requirements

### Requirement: A repo-root manifest declares multiple configurations

The system SHALL read a root `manifest.*` when the init location (local dir
or cloned git repo) contains no init file - the manifest is a list of
named configurations, each with an `init` path relative to the repo root and
optional matchers `os`, `distro`, `family`, `arch`, plus optional `default`.
The manifest SHALL decode strictly through the format registry.

Why: one blueprint repo commonly serves several machines (Arch/, macOS/,
Windows/…); today the operator must hand-point at the right subdirectory.

#### Scenario: Repo URL as init option

- **GIVEN** `--init-file` pointing at a git repo whose root has `manifest.yaml`
- **WHEN** `rwr all` runs
- **THEN** the repo is cloned via the existing blueprint git machinery and the
  manifest is read

### Requirement: Configuration selection matches detected OS

Manifest entries SHALL be filtered against detected OS info. Zero matches
SHALL error listing every entry and its matchers. Exactly one match SHALL be
used without prompting, logging which. Multiple matches SHALL present a TUI
selection frame before resolve stage 1.

#### Scenario: Single match auto-selected

- **GIVEN** an Arch host and a manifest whose only `family: arch` entry is
  `arch-desktop`
- **WHEN** `rwr all` runs
- **THEN** `arch-desktop` is used with no prompt and the choice is logged

#### Scenario: Multiple matches prompt

- **GIVEN** an Arch host and entries `arch-desktop` and `arch-server` both
  matching
- **WHEN** `rwr all` runs on a TTY
- **THEN** a selection frame lists both, matched entries first

### Requirement: Manifest paths cannot escape the repo

An entry's `init` path SHALL resolve inside the repo root; a path escaping it
SHALL be rejected before any file is read.

Why: blueprints (and their manifests) are untrusted input.

#### Scenario: Traversal rejected

- **GIVEN** an entry with `init: ../../etc/init.yaml`
- **WHEN** the manifest is validated
- **THEN** the entry is rejected with an error naming the path
