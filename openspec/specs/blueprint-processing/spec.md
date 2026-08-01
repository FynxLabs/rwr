# Blueprint Processing Specification

## Purpose

A blueprint declares what a machine should look like. This capability defines the
blueprint types, how files are discovered and ordered, how imports and profiles
narrow what applies, and how variables are resolved before anything runs.

## Requirements

### Requirement: Blueprints are written in YAML, JSON, or TOML

RWR SHALL read blueprints in YAML, JSON, and TOML, and SHALL treat the three as
interchangeable. The same declaration in any format SHALL produce the same result.

The tree declares its format in the init file, and RWR SHALL discover blueprint files
by that extension.

#### Scenario: The same blueprint in three formats

- **WHEN** an equivalent blueprint is written in YAML, JSON, and TOML
- **THEN** each decodes to the same structure and produces the same operations

### Requirement: There is one blueprint type per resource RWR manages

RWR SHALL support these blueprint types: `repositories`, `packages`, `ssh_keys`,
`users`, `files`, `fonts`, `services`, `git`, `scripts`, and `configuration`.

`directories` SHALL NOT be a blueprint type of its own. The `directories` key SHALL
be part of a files blueprint, alongside `files` and `templates`, and the files
processor SHALL read all three.

`packageManagers` SHALL NOT be a blueprint type. Package managers are installed from
the init file's own `packageManagers` section, ahead of the blueprint loop.

#### Scenario: A files blueprint with all three keys

- **WHEN** a files blueprint declares `directories`, `files`, and `templates`
- **THEN** the files processor acts on all three

#### Scenario: Package managers in the init file

- **WHEN** the init file declares `packageManagers`
- **THEN** they are installed before any blueprint type runs
- **AND** no processor is dispatched for a `packageManagers` blueprint type

### Requirement: The default run order covers every blueprint type

When the init file declares no order, RWR SHALL run blueprint types in this order:

1. `repositories`
2. `packages`
3. `ssh_keys`
4. `users`
5. `files`
6. `fonts`
7. `services`
8. `git`
9. `scripts`
10. `configuration`

`users` SHALL be present. Its previous absence meant `rwr all` never created users
unless the init file hand-wrote its own order, while `rwr run users` worked — which
made the omission easy to miss.

An init file that declares `blueprints.order` SHALL override this order.

#### Scenario: A default run

- **WHEN** `rwr all` runs against a tree whose init file declares no order
- **THEN** each declared blueprint type is dispatched exactly once, in the order above

#### Scenario: A tree with users but no declared order

- **WHEN** a tree contains a users blueprint and the init file declares no order
- **THEN** `rwr all` creates the users

#### Scenario: A custom order

- **WHEN** the init file declares an order of `repositories`, `packages`, `files`
- **THEN** only those three run, in that sequence

### Requirement: Bootstrap runs before the ordered blueprint types

When the blueprint tree contains a bootstrap file, RWR SHALL process it before the
ordered blueprint types.

RWR SHALL record that bootstrap ran and SHALL NOT repeat it, unless
`--force-bootstrap` is given.

#### Scenario: A second run on a bootstrapped machine

- **WHEN** `rwr all` runs on a machine that has already bootstrapped
- **THEN** bootstrap is skipped

#### Scenario: Forcing bootstrap

- **WHEN** `rwr all --force-bootstrap` runs
- **THEN** bootstrap runs again

### Requirement: Variables resolve before a blueprint is decoded

RWR SHALL render each blueprint file as a template before decoding it, exposing
`User`, `System`, `Flags`, and `UserDefined` variables.

`UserDefined` SHALL include every `RWR_`-prefixed environment variable, with the
prefix removed.

Template resolution SHALL apply to every blueprint type, including fonts.

#### Scenario: A blueprint referencing the current user

- **WHEN** a blueprint declares a target of `{{ .User.home }}/.config`
- **THEN** the path resolves to the running user's home directory

#### Scenario: A fonts blueprint with a variable

- **WHEN** a fonts blueprint references `{{ .UserDefined.fontDir }}`
- **THEN** the value is rendered before the fonts processor reads the blueprint

### Requirement: Any blueprint entry can import another file

RWR SHALL accept an `import` field on a blueprint entry, naming another blueprint
file whose definitions are merged in.

Import paths SHALL resolve relative to the importing blueprint's directory. An
imported file MAY be in any supported format. Imported entries SHALL be subject to
profile filtering like any other entry. Import SHALL work for every blueprint type.

RWR SHALL detect a circular import and SHALL NOT loop.

#### Scenario: Sharing a common package set

- **WHEN** a machine-specific blueprint imports `../../Common/packages/base.yaml`
- **AND** adds its own entries
- **THEN** both the imported and the local entries are applied

#### Scenario: A circular import

- **WHEN** file A imports B and B imports A
- **THEN** the cycle is detected and the run does not loop

#### Scenario: A nested import chain

- **WHEN** A imports B and B imports C
- **THEN** the definitions from all three files are applied

### Requirement: Profiles narrow what applies, permissively by default

RWR SHALL include a blueprint entry when any of these hold:

- The entry declares no profiles. Such an entry is a base item and always applies.
- No profiles are active. With no `--profile` given, everything applies.
- `all` is an active profile.
- One of the entry's profiles is active.

The permissive default exists so a tree works without the operator knowing anything
about profiles.

#### Scenario: A run with no profile flag

- **WHEN** `rwr all` runs with no `--profile`
- **THEN** every entry applies, profiled or not

#### Scenario: A run scoped to one profile

- **WHEN** `rwr all --profile work` runs
- **THEN** entries with no profiles apply
- **AND** entries listing `work` apply
- **AND** entries listing only `personal` do not

#### Scenario: The all profile

- **WHEN** `rwr all --profile all` runs
- **THEN** every entry applies

### Requirement: Blueprints may be cloned from a git repository

When the init file declares git options, RWR SHALL clone the blueprint repository to
the declared target, or update it in place when it is already a valid clone.

When the target exists but is not a git repository, RWR SHALL remove it and clone
fresh.

RWR SHALL fail when the resulting directory is empty.

#### Scenario: First run on a new machine

- **WHEN** the init file declares a blueprint repository and the target does not exist
- **THEN** the repository is cloned to the target

#### Scenario: A later run with updates enabled

- **WHEN** the target is a valid clone and updates are enabled
- **THEN** the repository is pulled before blueprints are processed

### Requirement: A missing blueprint file does not stop the run

RWR SHALL warn and continue when a file listed in the run order does not exist. RWR
SHALL stop when a blueprint that does exist fails to process.

#### Scenario: An order entry naming a removed file

- **WHEN** the init file's order names a file that is no longer present
- **THEN** RWR warns and processes the remaining blueprints

### Requirement: A missing template variable stops the run

RWR SHALL report an error when a blueprint references a variable that does not
exist, and SHALL NOT render a placeholder and continue.

An unresolved reference otherwise becomes a file path, a package name, or a service
name. Writing to `<no value>/.vimrc` is worse than refusing.

`rwr validate` SHALL be lenient about user-defined variables only, because it
cannot know which `RWR_*` variables the operator will export at run time. Every
other namespace SHALL be strict in both.

#### Scenario: A misspelled variable

- **WHEN** a blueprint declares `target: "{{ .User.Home }}/.vimrc"` and the key is
  `home`
- **THEN** the run stops and the error names the missing key

#### Scenario: Validating a tree that uses operator variables

- **WHEN** `rwr validate` runs on a tree referencing `{{ .UserDefined.company }}`
  and no such variable is exported
- **THEN** validation reports no error

#### Scenario: Running that same tree

- **WHEN** `rwr all` runs on it and no such variable is exported
- **THEN** the run stops rather than writing an empty value

### Requirement: Imports resolve through the whole chain

RWR SHALL follow imports that an imported file itself declares, to any depth.

An import path SHALL resolve relative to the file that declares it, so a chain can
walk into subdirectories.

RWR SHALL report a circular import rather than looping or silently applying part of
the graph. The same shared file reached from two different branches SHALL NOT be
treated as a cycle.

RWR SHALL report a missing import file.

RWR SHALL enforce the schema version of every file in the chain.

#### Scenario: A three-file chain

- **WHEN** A imports B and B imports C, each declaring one package
- **THEN** all three packages are installed

#### Scenario: A shared base reached twice

- **WHEN** two files both import the same base file
- **THEN** the base applies and no cycle is reported

#### Scenario: A genuine cycle

- **WHEN** A imports B and B imports A
- **THEN** the run stops with an error naming the circular import

#### Scenario: A newer schema two levels down

- **WHEN** a file three levels into an import chain declares an unsupported
  `schema_version`
- **THEN** the run stops and no command is issued
