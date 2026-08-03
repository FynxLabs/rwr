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
- A user or group action SHALL be valid as `create`, `modify`, `remove`, or
  `delete` — the last being the accepted alias for `remove`.
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

### Requirement: Validation checks declared file modes

`rwr validate` SHALL check the `mode` on every file, directory, and template entry:

- A `chmod` action with no mode SHALL be an error, because applying it would strip
  every permission off the target at run time.
- A mode larger than `0o7777` SHALL be an error: it is not a permission mode.
- A mode on an action that ignores it — `move`, `delete`, `chown`, `chgrp`,
  `symlink` — SHALL be a warning, so an operator who expected it to be applied
  finds out here. The mode-carrying actions are `create`, `chmod`, and `copy`:
  the files processor applies a declared mode to a copied target, so a mode on
  `copy` is meaningful, not ignored.
- A world-writable mode SHALL be a warning.
- A mode setting setuid or setgid SHALL be a warning.

An ambiguously written mode such as `mode: 644` never reaches these rules: it is
refused while the blueprint is being decoded, and arrives as a parse error naming
the file.

#### Scenario: A chmod with no mode

- **WHEN** a file entry declares `action: chmod` and no `mode`
- **THEN** validation reports an error and suggests a quoted octal string

#### Scenario: A mode on a symlink

- **WHEN** a file entry declares `action: symlink` and `mode: "0644"`
- **THEN** validation warns that the mode is ignored by that action

#### Scenario: A world-writable mode

- **WHEN** a file entry declares `mode: "0666"`
- **THEN** validation warns about the world-write bit

### Requirement: Unknown keys are rejected when decoding strictly

Strict decoding SHALL reject a blueprint containing a key the schema does not
define, across YAML, JSON, and TOML. It SHALL be what a run uses as well as what
validation uses, so `rwr validate` and `rwr all` agree on which keys exist; see the
blueprint-processing specification.

A silently ignored key is a blueprint that looks applied and is not — a misspelled
`profiles` key means an entry runs on every machine instead of the one it was
scoped to.

The `schema_version` probe SHALL remain lenient: it reads a single key out of a
full document to decide which schema to decode against, so the other keys are not
its concern.

#### Scenario: A misspelled key

- **WHEN** a blueprint declares `packagess:` instead of `packages:`
- **THEN** strict decoding reports the unknown key

#### Scenario: A misspelled entry key

- **WHEN** a package entry declares `profile:` instead of `profiles:`
- **THEN** strict decoding reports the unknown key

#### Scenario: An empty document

- **WHEN** a blueprint file contains no keys at all
- **THEN** strict decoding accepts it as a section with nothing in it

### Requirement: Template strictness at validate matches the run

`rwr validate` SHALL resolve template references strictly for the `User`,
`System`, and `Flags` namespaces — a reference that does not exist is a
validation error — and leniently (`missingkey=zero`) only for `UserDefined`,
whose values legitimately vary per machine.

Why: validate resolved every namespace leniently, so a typo like
`{{ .User.hoem }}` validated clean and failed at run time — the exact class of
error validate exists to catch early.

#### Scenario: Misspelled built-in reference

- **WHEN** a blueprint references `{{ .User.hoem }}`
- **THEN** `rwr validate` reports it as an error naming the reference
- **AND** an undefined `{{ .UserDefined.anything }}` still validates

### Requirement: Declaring both name and names is flagged

An entry declaring both `name` and `names` SHALL produce a validation warning
naming the entry, since only the `names` list will be processed.

#### Scenario: Both declared

- **WHEN** a packages entry declares both `name` and `names`
- **THEN** validate warns that `name` is ignored

### Requirement: Embedded provider contracts live in the CUE schema

Embedded provider contracts SHALL be expressed in the CUE schema — the checks
currently hand-rolled in `internal/validate/providers.go`: required `name`,
`detection.binary`, `commands.install`; step `action` constrained to the enum
the processors implement; `condition` restricted to derivable predicate names;
no literal `/tmp/` paths in steps. Go validation SHALL remain only for
filesystem overrides.

Why: the hand-rolled Go checks have already drifted from the processors once
(fictional actions, stale fields); a schema the export gate enforces cannot
drift silently.

#### Scenario: Invalid action rejected at export

- **GIVEN** a CUE provider step with `action: "instal"`
- **WHEN** the export runs
- **THEN** it fails listing the allowed actions

#### Scenario: Predicate list cannot drift

- **GIVEN** the CUE `or` list of condition predicates and Go's
  `repositoryPredicates` keys
- **WHEN** the cross-check test runs
- **THEN** it fails if the two sets differ in either direction

### Requirement: CUE errors are validate diagnostics

`rwr validate` SHALL evaluate `.cue` blueprints and report evaluation and
unification failures as diagnostics with file and line, on the same surface
as schema errors for the other formats.

#### Scenario: Constraint violation reported

- **GIVEN** a `.cue` blueprint violating one of its own constraints
- **WHEN** `rwr validate` runs
- **THEN** the diagnostic names the file, line, and failed constraint, and
  validation exits nonzero

### Requirement: Examples cover CUE

`examples/` SHALL cover every blueprint type in CUE as a fourth format column,
validated in CI, per the compatibility contract.

#### Scenario: CI validates CUE examples

- **GIVEN** the examples tree
- **WHEN** CI runs
- **THEN** every `.cue` example validates like its YAML/JSON/TOML siblings

## Known Gaps

- **A blueprint tree with no init file cannot be validated.** `rwr validate` looks
  for an init file at or above the path it is given and reports an error without
  one, so a directory of blueprint files on its own cannot be checked.
  `examples/imports/` is in that state today; CI pins the set of unreachable
  example directories so it cannot grow silently.
