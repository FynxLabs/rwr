# Blueprint Validation Specification

## Purpose

`rwr validate` is the check an operator runs before letting RWR touch a machine, and
the check CI runs to catch a change that breaks trees already in the wild. This
capability defines what validation inspects and what it reports.

## Requirements

### Requirement: Validation reports issues by severity and fails on errors

RWR SHALL classify each validation issue as error, warning, or info, and SHALL report
counts for each.

`rwr validate` SHALL exit non-zero when any error is found, and SHALL succeed with a
count when only warnings are found.

#### Scenario: A tree with a structural error

- **WHEN** `rwr validate` runs against a tree containing an invalid blueprint
- **THEN** the command exits non-zero and reports the error and warning counts

#### Scenario: A clean tree

- **WHEN** `rwr validate` runs against a valid tree
- **THEN** the command succeeds and reports success

### Requirement: Validation chooses blueprint or provider mode from the path

RWR SHALL validate as providers when the target is a directory named `providers` or
a single `.toml` file, and as blueprints otherwise.

RWR SHALL accept `--blueprints` and `--providers` to force either mode.

#### Scenario: Validating a provider file

- **WHEN** `rwr validate providers/paru.toml` runs
- **THEN** the file is checked as a provider definition

#### Scenario: Forcing blueprint mode

- **WHEN** `rwr validate path/to/dir --blueprints` runs on a directory
- **THEN** the contents are checked as blueprints

### Requirement: Validation runs without initializing the system

`rwr validate` SHALL detect the operating system and set paths, but SHALL NOT load
the init file, clone the blueprint repository, or perform any other initialization.

Validation is what an operator runs when something is wrong; it must not require the
thing being validated to already work.

#### Scenario: Validating a tree with a broken init file

- **WHEN** `rwr validate` runs against a tree whose init file has a structural error
- **THEN** the error is reported rather than causing the command to fail during
  startup

### Requirement: The shipped examples validate in CI

The example blueprints SHALL be checked on every pull request. The check SHALL cover
every example, in every supported format, for every platform directory.

The examples are the backwards-compatibility contract: a change that stops an
example from parsing, rendering, or decoding is a change that breaks trees in the
wild, and CI is where that must surface.

#### Scenario: A change that breaks a blueprint format

- **WHEN** a change alters how a blueprint field is read
- **AND** a shipped example uses that field
- **THEN** CI fails on the examples check

#### Scenario: Cross-format agreement

- **WHEN** the same example is provided in YAML, JSON, and TOML
- **THEN** all three decode to the same structure

#### Scenario: An unresolved template variable

- **WHEN** an example references a variable that does not resolve
- **THEN** the check fails rather than rendering `<no value>` into the output

### Requirement: Validation inspects the whole tree

`rwr validate` SHALL walk the blueprint tree and check every blueprint file, not
only the files in the top directory.

Blueprints are organised by type — `packages/`, `files/`, `services/` — which is the
layout the documentation recommends and every example uses. Reading only the top
directory means checking the init file and reporting success.

RWR SHALL skip dot directories, and SHALL NOT validate a nested init file as a
blueprint.

#### Scenario: A tree organised by type

- **WHEN** `rwr validate` runs on a tree with an invalid action in
  `packages/dev.yaml`
- **THEN** the invalid action is reported

#### Scenario: A tree under version control

- **WHEN** the blueprint tree contains a `.git` directory
- **THEN** nothing under it is validated

### Requirement: Validation reads blueprints the way a run does

`rwr validate` SHALL render each blueprint as a template before decoding it, decode
it into the same structure the matching processor uses, and enforce the same schema
version.

Anything else validates a different document than the one that will be applied. A
blueprint using `{{ .User.home }}` is not valid YAML until it is rendered, because
the braces read as a flow mapping.

#### Scenario: A blueprint using a variable

- **WHEN** a files blueprint declares `target: "{{ .User.home }}/.vimrc"`
- **THEN** validation reports no error

#### Scenario: A blueprint declaring an unsupported schema version

- **WHEN** a blueprint declares a `schema_version` this build cannot read
- **THEN** validation reports it, rather than passing a tree a run will refuse

### Requirement: Validation accepts every form the processors accept

A validation rule SHALL accept exactly what the matching processor accepts. A rule
that rejects what a processor accepts is a false report, and a rule that accepts
what a processor rejects is a missed one. Specifically:

- A package entry SHALL be valid with `name` or with `names`.
- `package_manager` SHALL be optional; without one the default manager applies.
- A file entry's action SHALL be one of `create`, `delete`, `copy`, `move`,
  `chmod`, `chown`, `chgrp`, `symlink` — the set the files processor dispatches on.
- A script SHALL be valid with `exec`, `content`, or `source`.
- Every blueprint type SHALL have a validator, including `fonts` and
  `configuration`.

#### Scenario: A multi-package entry

- **WHEN** a package entry declares `names: [curl, wget]` and no `name`
- **THEN** validation reports no error

#### Scenario: A copied dotfile

- **WHEN** a file entry declares `action: copy`
- **THEN** validation reports no error

#### Scenario: A script from a file

- **WHEN** a script declares `source` and no `exec` or `content`
- **THEN** validation reports no error

#### Scenario: An entry naming nothing

- **WHEN** a package entry declares neither `name` nor `names`
- **THEN** validation reports an error

### Requirement: Unknown keys are rejected when decoding strictly

Strict decoding SHALL reject a blueprint containing a key the schema does not
define, across YAML, JSON, and TOML.

A silently ignored key is a blueprint that looks applied and is not — a misspelled
`profiles` key means an entry runs on every machine instead of the one it was
scoped to.

#### Scenario: A misspelled key

- **WHEN** a blueprint declares `packagess:` instead of `packages:`
- **THEN** strict decoding reports the unknown key

## Known Gaps

- **Strict decoding is not enabled.** The requirement above describes the intended
  contract; the shipped decoders still ignore unknown keys.
