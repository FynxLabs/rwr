# CLI — Deltas

## ADDED Requirements

### Requirement: rwr diff reports machine drift as blueprint material

`rwr diff` SHALL compare the system scan against the resolved blueprint
tree and report additions (on the machine, absent from the tree) and
removals (declared, gone), grouped by category and provider. Packages the
journal shows a run applied SHALL NOT report as hand-added. With
`--format`, the delta SHALL render as paste-ready blueprint blocks in the
named registry format. Diff never mutates the system.

#### Scenario: Hand-installed package surfaces

- **GIVEN** a package present in the explicit list, absent from the tree
  and from the journal
- **WHEN** `rwr diff --packages` runs
- **THEN** it is reported as an addition with its provider

#### Scenario: Blueprint-installed package is not drift

- **GIVEN** a package the journal records a run applied
- **WHEN** `rwr diff` runs
- **THEN** it is not reported as an addition

#### Scenario: Paste-ready output

- **WHEN** `rwr diff --packages --format cue` runs
- **THEN** the output is a packages block that strict-decodes

### Requirement: rwr diff --into routes changes into the tree interactively

`rwr diff --into <tree>` SHALL offer each change group a destination chosen
by the operator — the matching machine tree's file for that category, a
Common file the tree imports, or skip — and SHALL write accepted edits in
the destination file's own format, leaving a tree that passes validation.
Whether a change is machine-specific or Common is the operator's decision;
rwr's job is to make it once per group instead of once per hand-edit.

#### Scenario: Routed into Common

- **GIVEN** a new package and a tree whose packages file imports a Common
  base
- **WHEN** the operator picks the Common destination
- **THEN** the entry lands in the Common file, in that file's format, and
  the tree validates

#### Scenario: Non-interactive refuses with the alternative

- **WHEN** `rwr diff --into tree` runs without a TTY
- **THEN** it fails naming `--format` as the non-interactive path
