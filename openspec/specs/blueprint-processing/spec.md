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

`packageManagers` SHALL NOT appear in this list: it is not dispatched from the
blueprint loop at all, and listing it only produced an "Unknown processor" warning.

When an `order` entry is a mapping rather than a plain name, RWR SHALL run the named
processors in sorted order and SHALL warn that it did so, naming them. A mapping
cannot preserve the order it was written in, and Go randomizes map iteration, so
such an entry previously produced a different sequence on every run — the one thing
an explicit order is written to control.

#### Scenario: A nested order mapping

- **WHEN** an `order` entry is a mapping naming `packages` and `files`
- **THEN** they run in sorted order
- **AND** RWR warns naming the sorted sequence and suggesting a flat list

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

RWR SHALL look for the bootstrap file in every supported format — `bootstrap.yaml`,
`bootstrap.yml`, `bootstrap.json`, `bootstrap.toml`. Looking only for
`bootstrap.yaml` meant a tree written in TOML or JSON silently never bootstrapped,
with no message saying so.

RWR SHALL record that bootstrap ran and SHALL NOT repeat it, unless
`--force-bootstrap` is given.

#### Scenario: A TOML tree with a bootstrap file

- **WHEN** a tree declares `format: toml` and contains `bootstrap.toml`
- **THEN** bootstrap is processed before the ordered blueprint types

#### Scenario: A second run on a bootstrapped machine

- **WHEN** `rwr all` runs on a machine that has already bootstrapped
- **THEN** bootstrap is skipped

#### Scenario: Forcing bootstrap

- **WHEN** `rwr all --force-bootstrap` runs
- **THEN** bootstrap runs again

### Requirement: Blueprints are decoded strictly on a run, not only in tests

Every blueprint a processor reads SHALL be decoded with unknown keys rejected, in
YAML, JSON, and TOML alike. `helpers.DecodeBlueprint` SHALL use the strict decoder,
and every processor SHALL read its blueprint through `DecodeBlueprintInto`.

A silently ignored key is a blueprint that looks applied and is not: `pacakges:`
yields an empty section, every processor finds nothing to do, and the run reports
success having changed nothing. A misspelled `profiles` is worse — the entry loses
its scoping and runs on every machine. Both are invisible at any log level, so they
surface as "rwr didn't do anything" long after the typo was written.

The `schema_version` probe SHALL remain lenient, because it reads a single key out
of a whole document to decide which schema to decode against.

#### Scenario: A misspelled top-level key on a real run

- **WHEN** `rwr all` reads a blueprint declaring `packagess:`
- **THEN** the run stops with an error naming the unknown key

#### Scenario: An empty blueprint file

- **WHEN** a blueprint file contains no keys at all
- **THEN** it decodes as a section with nothing in it and the run continues

### Requirement: Applying a state that already holds is not a failure

RWR SHALL check for the desired state before issuing a mutating command, and SHALL
converge rather than abort when that state already holds. Running `rwr all` a second
time on an unchanged machine is the normal case, not an error path. Specifically:

- A `create` for a user or group that already exists SHALL converge the declared
  attributes instead of running `useradd`/`groupadd`, which exit non-zero on
  "already exists" and used to abandon the rest of the run.
- A `remove` for a user or group that does not exist SHALL report that there is
  nothing to remove and succeed.
- A symlink already pointing at the declared source SHALL be left alone; one
  pointing elsewhere SHALL be replaced, with the change logged; a regular file or
  directory in the way SHALL be an error rather than a silent replacement.
- Deleting a file or directory that is already absent SHALL succeed.
- A move whose source is gone because an earlier run performed it SHALL succeed.

#### Scenario: Running the same tree twice

- **WHEN** `rwr all` runs twice against an unchanged tree
- **THEN** the second run makes no change and reports no error

#### Scenario: A symlink that points somewhere else

- **WHEN** the declared symlink target exists and points at a different source
- **THEN** RWR logs the old and new source and repoints the link

#### Scenario: A symlink target occupied by a real file

- **WHEN** a regular file already exists at the declared symlink target
- **THEN** RWR reports an error rather than removing it

### Requirement: Non-fatal item failures reach the exit code

A processor that SHALL continue past a failed item — a package that is not in the
repositories, a git remote that is temporarily unreachable, an SSH key that could
not be registered — SHALL record that failure in a run ledger rather than only
logging it.

At the end of the run RWR SHALL report every recorded failure and SHALL return an
error, so the exit code is non-zero.

Without the ledger those failures vanished: they were logged, the run printed
"RWR Run Complete!" and exited 0, so a run in which every single package failed to
install was indistinguishable from a clean one.

The ledger is populated by the packages, git, and `ssh_keys` processors.

#### Scenario: A run where every package fails

- **WHEN** every package in a tree fails to install
- **THEN** the remaining blueprint types still run
- **AND** the run ends by listing the failures and exiting non-zero

#### Scenario: A clean run

- **WHEN** nothing fails
- **THEN** RWR reports the run complete and exits zero

### Requirement: A file mode is declared unambiguously

A blueprint SHALL declare a file or directory mode as a quoted octal string —
`"0644"`, `"644"`, `"0o644"` — which is the recommended form because it means the
same thing in YAML, JSON, and TOML.

A number SHALL be read as the mode's own value, which is what every parser already
produces for an octal literal: YAML `0644`, TOML `0o644`, and JSON `420` are one
mode. A bare decimal that instead reads like unquoted octal digits — `644`, `755` —
SHALL be refused with an error showing both readings, rather than guessed at.

A mode SHALL NOT exceed `0o7777`. Zero SHALL mean "no mode declared", so a
processor can apply its default.

Where no mode is declared, a rendered template SHALL be written `0600` and a plain
file `0644`. A template's output is the place a credential is most likely to land.

#### Scenario: A quoted octal mode

- **WHEN** a file entry declares `mode: "0644"` in YAML, JSON, or TOML
- **THEN** the file is created with mode `0644` in all three

#### Scenario: An unquoted decimal that reads as octal

- **WHEN** a file entry declares `mode: 644`
- **THEN** the blueprint is refused with an error naming the file and showing both
  the value `0o1204` and the intended `0o644`

#### Scenario: A rendered template with no mode

- **WHEN** a templates entry declares no `mode`
- **THEN** the rendered file is written at `0600`

### Requirement: Repository action steps are rendered before they run

RWR SHALL render every templated field of a repository action step against the
repository's values before acting on it, with `missingkey=error`. A provider's
`add` and `remove` steps are Go templates — apt writes
`deb [arch={{ .Arch }} signed-by={{ .KeyPath }}] {{ .URL }} ...`.

Previously the placeholders were written to disk literally, producing source files
containing `{{ .URL }}`.

A step MAY declare a `condition`, which SHALL be evaluated before the rest of the
step is rendered — a skipped step is allowed to reference data this repository does
not carry, which is the reason it is conditional. Only an explicitly truthy
rendering SHALL run the step.

RWR SHALL support these step actions: `exec`/`command`, `download`, `write`,
`copy`, `append`, `remove_line`, `remove_section`, and `remove`. Any other action
SHALL stop the run.

`action: remove` on a repository blueprint SHALL run the provider's remove steps.
Every embedded provider defines them.

The steps that edit or delete an existing file — `append`, `remove_line`,
`remove_section`, and `remove` — SHALL resolve their path against the provider's
declared repository directories and SHALL refuse a path that resolves outside them,
including through a symlink.

#### Scenario: Adding an apt repository

- **WHEN** an apt repository is added
- **THEN** the written source file contains the resolved URL and key path, not the
  template placeholders

#### Scenario: A step whose condition does not hold

- **WHEN** a step declares `condition = "{{ .HasKey }}"` and the repository has no key
- **THEN** the step is skipped and its other fields are never rendered

#### Scenario: Removing a repository

- **WHEN** a repositories blueprint declares `action: remove`
- **THEN** the provider's remove steps run — deleting the source file and keyring,
  or removing the named section from the provider's config file

#### Scenario: A step path outside the provider's directories

- **WHEN** a rendered `remove` or `append` path resolves outside the provider's
  declared repository paths
- **THEN** the run stops rather than removing or appending there

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

Import paths SHALL resolve relative to the file that declares them, in every
processor. (Previously six processors resolved top-level imports against the
tree root while four resolved file-relative — the same `import:` string meant
two different files depending on the blueprint type.) An imported file MAY be
in any supported format, and its format SHALL be derived from its own
extension, not the importing file's. Imported entries SHALL be subject to
profile filtering like any other entry. Import SHALL work for every blueprint
type.

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

#### Scenario: Same relative import in two blueprint types

- **GIVEN** `packages/base.yaml` and `files/base.yaml`, each declaring
  `import: ../shared/common.yaml`
- **WHEN** both are processed
- **THEN** both resolve the same file, relative to each importing file

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

When the target exists but is not a git repository, RWR SHALL refuse with an error
naming the path and SHALL NOT delete it. RWR runs unattended, and a mistyped target
such as `~/dotfiles` was previously reclaimed by deleting whatever was there.

RWR SHALL fail when the resulting directory is empty.

In dry-run mode RWR SHALL report the sync it would perform and SHALL touch neither
disk nor network, because cloning and pulling go directly to the filesystem rather
than through the command executor where dry-run is otherwise enforced.

#### Scenario: First run on a new machine

- **WHEN** the init file declares a blueprint repository and the target does not exist
- **THEN** the repository is cloned to the target

#### Scenario: A later run with updates enabled

- **WHEN** the target is a valid clone and updates are enabled
- **THEN** the repository is pulled before blueprints are processed

#### Scenario: A target that already holds something else

- **WHEN** the declared target exists and is not a git repository
- **THEN** the run stops with an error naming the path
- **AND** nothing at that path is removed

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

### Requirement: Downloads are validated on every hop, bounded, and pinnable

A download URL SHALL be refused unless it is https (plain http is allowed only
for loopback hosts) — everything RWR downloads (package-signing keys, fonts,
file sources, the init file) is installed with the operator's privileges — and
that check SHALL be re-applied to **every redirect hop**, not only the initial
URL. Validating only the first URL is worthless the moment a server answers
with a 302 to plain http.

Downloads SHALL time out rather than hang a run indefinitely.

Where a step or blueprint entry declares a `sha256` for downloaded content, RWR
SHALL verify the download against it before the content is moved into place,
and SHALL discard the download on a mismatch. This applies to repository action
steps and package-manager install/remove steps alike.

A repository MAY declare `key_sha256`, the sha256 of the signing key at
`key_url`; when declared, the provider's key-download step SHALL verify it.
A `files:` entry with a URL source MAY declare `sha256`; when declared, the
download SHALL be verified before the file is installed.

A repository with `key_url` and no `key_sha256`, and a files entry with a URL
source and no `sha256`, SHALL produce a prominent warning naming the unpinned
download. (A later major version refuses; the policy ratchets, never loosens.)

#### Scenario: A redirect to plain http

- **WHEN** an https download URL answers with a redirect to a plain-http,
  non-loopback URL
- **THEN** the download is refused before any request is made to the redirect
  target

#### Scenario: An install step with a wrong digest

- **WHEN** a provider's install step declares `sha256` and the downloaded
  content does not match it
- **THEN** the step fails, the staged download is discarded, and nothing is
  written to the destination

#### Scenario: Pinned signing key mismatch

- **WHEN** a repository declares `key_url` and `key_sha256` and the downloaded
  key does not match
- **THEN** the repository is not added and the failure names the digest
  mismatch

#### Scenario: Unpinned signing key warns

- **WHEN** a repository declares `key_url` without `key_sha256`
- **THEN** the run proceeds with a prominent warning naming the repository

### Requirement: `names` wins over `name` everywhere

When an entry declares both `name` and `names`, the `names` list SHALL be
processed and `name` ignored, for every blueprint type. (packages had it
backwards relative to files/fonts.)

#### Scenario: Both declared on a packages entry

- **WHEN** a packages entry declares `name: git` and `names: [vim, tmux]`
- **THEN** vim and tmux are installed and git is not

### Requirement: Blueprint type is derived from content when the path names none

A blueprint file whose path names no processor directory SHALL be typed by its
top-level keys (`packages:` → packages, `repositories:` → repositories, …).
A file whose content matches no type — or more than one — SHALL produce the
loud unrouted-file warning and SHALL NOT execute.

Why: the flattened and minimal_files layouts the examples ship were dead ends —
path-only dispatch sent every file to a bucket the run loop never reads, and
the run exited 0 having executed nothing.

#### Scenario: Flattened layout executes

- **GIVEN** a tree with `packages.yaml` at its root declaring packages
- **WHEN** `rwr all` runs
- **THEN** the file is dispatched to the packages processor and executes

#### Scenario: Ambiguous content still warns

- **GIVEN** a root-level file declaring both `packages:` and `services:`
- **WHEN** the run order is computed
- **THEN** the file is not executed and a warning names it and why

### Requirement: Format registry is the single source of blueprint format knowledge

The system SHALL resolve a blueprint file's format from its extension through a
single registry. No other code SHALL enumerate blueprint extensions or map an
extension to a decoder.

Why: format knowledge is duplicated across ~15 sites today; each new format is
N hand edits, and the copies have already diverged (see next requirement).

#### Scenario: Unknown extension yields a diagnostic, not a panic

- **GIVEN** a blueprint directory containing `packages.xml`
- **WHEN** the run order is processed
- **THEN** the file is reported as an unsupported format with its path
- **AND** the run does not panic (today `all.go:148` panics on an extensionless file)

### Requirement: Format is derived per file

The system SHALL determine each blueprint file's format from that file's
extension. `Init.Format` SHALL NOT cause a file with a recognized extension to
be decoded as another format.

Why: half the codebase assumes a tree-uniform format via `Init.Format`, half
derives per file; a mixed tree behaves differently depending on which code path
touches it.

#### Scenario: Mixed-format tree

- **GIVEN** a blueprint tree containing `packages.yaml` and `files.toml`
- **WHEN** `rwr all` processes the tree
- **THEN** each file is decoded according to its own extension
- **AND** profile discovery and file ordering see both files

### Requirement: CUE is a supported blueprint format

`.cue` SHALL be a registered blueprint format alongside YAML, JSON, and TOML,
valid for blueprints, imports, init files, bootstrap files, and the manifest.
A `.cue` file SHALL be evaluated in-process (`cuelang.org/go`), exported to
concrete values, and decoded through the existing strict path — semantics
identical to the equivalent YAML.

Why: CUE gives authors types, constraints, and composition at authoring time;
errors surface before a run touches the machine. Evaluation is in-process
because rwr bootstraps machines that do not have a `cue` binary.

#### Scenario: CUE blueprint equals its YAML twin

- **GIVEN** `packages.cue` and `packages.yaml` declaring the same packages
- **WHEN** both are decoded
- **THEN** the resulting blueprint values are identical

#### Scenario: Non-concrete value is an error

- **GIVEN** a `.cue` blueprint with an unresolved field (`version: string`)
- **WHEN** it is evaluated
- **THEN** decoding fails with the field name and file/line

### Requirement: CUE evaluation is sandboxed to the blueprint tree

CUE evaluation SHALL NOT resolve modules, imports, or embeds from the
network or from filesystem paths outside the blueprint tree.

Why: blueprints are untrusted input; evaluation must not become a way to read
arbitrary files or phone home.

#### Scenario: Escape rejected

- **GIVEN** a `.cue` file importing a path outside the blueprint tree
- **WHEN** it is evaluated
- **THEN** evaluation fails with a containment error

## Known Gaps

- **The failure ledger covers four processors.** `packages`, `git`, `ssh_keys` and
  `configuration` record their skipped items. The files, services, repositories and
  fonts processors abort the whole run on a failure instead, which does reach the
  exit code but gives up the remaining work rather than reporting it at the end.
- **Path containment is only applied to repository steps that edit an existing
  file.** The `download`, `write` and `copy` repository steps write to the
  destination a provider names with no containment check, and the files processor's
  own `target` is not contained either.
- **Windows users are unimplemented.** `ProcessUsers` has Linux (shadow-utils) and
  macOS (Open Directory) implementations; on Windows it logs a warning per entry
  and does nothing.
