# CLI - Deltas

## ADDED Requirements

### Requirement: rwr convert converts trees and migrates deprecated constructs

`rwr convert [path]` SHALL convert every blueprint, init, bootstrap, and
manifest file in a tree to another format (`--to yaml|json|toml|cue`) and
SHALL rewrite deprecated constructs to their current equivalents
(`--migrate`). It SHALL be a dry run by default: nothing is written without
`--write`.

Comments are not preserved across formats; the command SHALL warn per file
that carries them. A file whose template placeholders make it unparseable
SHALL be reported and skipped, never mangled. CUE output is JSON-form CUE:
valid and lossless; idiomatic CUE remains authoring work.

The first migration rule SHALL move init-file inline resource sections
(`repositories`, `packages`, `services`, `files`, `templates`, `directories`,
`configuration`) out of the init file into blueprint files under the tree -
the construct strict decode now rejects.

#### Scenario: Dry run by default

- **WHEN** `rwr convert tree --to toml` runs without `--write`
- **THEN** the planned conversions are printed and no file is created,
  removed, or modified

#### Scenario: Tree conversion with --write

- **GIVEN** a YAML tree
- **WHEN** `rwr convert tree --to toml --write` runs
- **THEN** each blueprint is rewritten as `.toml`, the original files are
  removed, and the resulting tree validates

#### Scenario: Migrating inline init sections

- **GIVEN** an init file declaring a top-level `packages:` list
- **WHEN** `rwr convert tree --migrate --write` runs
- **THEN** the section lands in `packages/from-init.<ext>`, the init file
  keeps only its own keys, and a re-run reports nothing to change

#### Scenario: Unparseable templated file is skipped

- **WHEN** a blueprint's template placeholders make it unparseable
- **THEN** the file is reported and left untouched
