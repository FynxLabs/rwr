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
unless the init file hand-wrote its own order, while `rwr run users` worked - which
made the omission easy to miss.

An init file that declares `blueprints.order` SHALL override this order.

`packageManagers` SHALL NOT appear in this list: it is not dispatched from the
blueprint loop at all, and listing it only produced an "Unknown processor" warning.

When an `order` entry is a mapping rather than a plain name, RWR SHALL run the named
processors in sorted order, and when the mapping names more than one processor RWR
SHALL warn that it did so, naming them. A mapping cannot preserve the order it was
written in, and Go randomizes map iteration, so such an entry previously produced a
different sequence on every run - the one thing an explicit order is written to
control. A mapping naming a single processor has only one possible order, so it
produces no warning.

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

RWR SHALL look for the bootstrap file in every supported format - `bootstrap.yaml`,
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
success having changed nothing. A misspelled `profiles` is worse - the entry loses
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

A processor that SHALL continue past a failed item - a package that is not in the
repositories, a git remote that is temporarily unreachable, an SSH key that could
not be registered - SHALL record that failure in a run ledger rather than only
logging it.

At the end of the run RWR SHALL report every recorded failure and SHALL return an
error, so the exit code is non-zero.

Without the ledger those failures vanished: they were logged, the run printed
"RWR Run Complete!" and exited 0, so a run in which every single package failed to
install was indistinguishable from a clean one.

The ledger is populated by the packages, repositories, services, files (including
its templates and directories sections), configuration, git, `ssh_keys`, and fonts
processors. The users and scripts processors do not use it; they return an error on
the first failed entry instead (see Known Gaps).

#### Scenario: A run where every package fails

- **WHEN** every package in a tree fails to install
- **THEN** the remaining blueprint types still run
- **AND** the run ends by listing the failures and exiting non-zero

#### Scenario: A clean run

- **WHEN** nothing fails
- **THEN** RWR reports the run complete and exits zero

### Requirement: A file mode is declared unambiguously

A blueprint SHALL declare a file or directory mode as a quoted octal string -
`"0644"`, `"644"`, `"0o644"` - which is the recommended form because it means the
same thing in YAML, JSON, and TOML.

A number SHALL be read as the mode's own value, which is what every parser already
produces for an octal literal: YAML `0644`, TOML `0o644`, and JSON `420` are one
mode. A bare decimal that instead reads like unquoted octal digits - `644`, `755` -
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
`add` and `remove` steps are Go templates - apt writes
`deb [arch={{ .Arch }} signed-by={{ .KeyPath }}] {{ .URL }} ...`.

Previously the placeholders were written to disk literally, producing source files
containing `{{ .URL }}`.

A step MAY declare a `condition`, which SHALL be evaluated before the rest of the
step is rendered - a skipped step is allowed to reference data this repository does
not carry, which is the reason it is conditional. Only an explicitly truthy
rendering SHALL run the step.

RWR SHALL support these step actions: `exec`/`command`, `download`, `write`,
`copy`, `append`, `remove_line`, `remove_section`, and `remove`. Any other action
SHALL stop the run.

`action: remove` on a repository blueprint SHALL run the provider's remove steps.
Every embedded provider defines them.

The steps that edit or delete an existing file - `append`, `remove_line`,
`remove_section`, and `remove` - SHALL resolve their path against the provider's
declared repository directories and SHALL refuse a path that resolves outside them,
including through a symlink.

The steps that create a file - `download`, `write`, and `copy` - SHALL resolve
their destination the same way, accepting only a destination inside the provider's
declared repository paths or the run's private staging directory. These steps run
with the provider's privileges - root, for every system package manager - and the
destination is a template rendered against blueprint values, so an unchecked
destination would be a root-privileged write to an arbitrary path.

File creation and replacement SHALL reject a final destination that is a symlink
or Windows reparse point. Permission changes SHALL be applied through the opened
destination descriptor so a path replacement cannot redirect the change.

#### Scenario: Adding an apt repository

- **WHEN** an apt repository is added
- **THEN** the written source file contains the resolved URL and key path, not the
  template placeholders

#### Scenario: A step whose condition does not hold

- **WHEN** a step declares `condition = "{{ .HasKey }}"` and the repository has no key
- **THEN** the step is skipped and its other fields are never rendered

#### Scenario: Removing a repository

- **WHEN** a repositories blueprint declares `action: remove`
- **THEN** the provider's remove steps run - deleting the source file and keyring,
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
tree root while four resolved file-relative - the same `import:` string meant
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

### Requirement: An unknown profile name is refused

RWR SHALL validate the names given to `--profile` against the profiles the
blueprint tree declares, before processing begins. A name no blueprint declares
SHALL stop the run with an error listing the available profiles.

When the tree declares no profiles at all, RWR SHALL warn that every
profile-scoped entry will be skipped and continue - an empty discovery is as
likely to mean the walk missed something as that the operator mistyped. When
profile discovery itself fails, RWR SHALL continue without validating; the
processors will report why with better context.

A misspelled profile name used to be silent: the filter matched nothing, every
profile-scoped entry was skipped, and the run reported success having installed
only the base items. A mistyped profile looked exactly like a working run.

#### Scenario: A mistyped profile name

- **WHEN** `rwr all --profile wrok` runs against a tree declaring `work` and `personal`
- **THEN** the run stops with an error naming `wrok` and listing the available profiles

#### Scenario: A profile flag against a profile-free tree

- **WHEN** `--profile work` is given and no blueprint in the tree declares any profiles
- **THEN** RWR warns that every profile-scoped entry will be skipped
- **AND** the run continues

### Requirement: Blueprints may be cloned from a git repository

When the init file declares git options, RWR SHALL clone the blueprint repository to
the declared target, or update it in place when it is already a valid clone.

When the target exists but is not a git repository, RWR SHALL refuse with an error
naming the path and SHALL NOT delete it. RWR runs unattended, and a mistyped target
such as `~/dotfiles` was previously reclaimed by deleting whatever was there.

RWR SHALL fail when the resulting directory is empty.

In dry-run mode RWR SHALL report the sync it would perform and SHALL touch neither
disk nor network, because cloning and pulling go directly to the filesystem rather
than through the command executor where dry-run is otherwise enforced. This
includes not creating the declared clone target or any parent directories.

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

### Requirement: A managed clone's origin follows the declared URL

RWR SHALL compare a managed clone's `origin` remote URL against the URL the
blueprint declares - for a `git` blueprint entry that already exists on disk,
and for the blueprint repository itself - and SHALL re-point `origin` to the
declared URL when they differ, before any pull.

The blueprint is the source of truth for where a repository comes from; without
the re-point, editing a URL in a blueprint changed nothing on machines that had
already cloned.

#### Scenario: A repository whose declared URL changed

- **WHEN** a git entry's declared URL differs from the existing clone's `origin`
- **THEN** `origin` is re-pointed to the declared URL
- **AND** the subsequent pull uses the new URL

#### Scenario: A matching origin

- **WHEN** the declared URL and the existing `origin` already match
- **THEN** the remote is left untouched

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
for loopback hosts) - everything RWR downloads (package-signing keys, fonts,
file sources, the init file) is installed with the operator's privileges - and
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
A file whose content matches no type - or more than one - SHALL produce the
loud unrouted-file warning and SHALL NOT execute.

Why: the flattened and minimal_files layouts the examples ship were dead ends -
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
concrete values, and decoded through the existing strict path - semantics
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
### Requirement: Configuration entries apply desktop settings through named tools

RWR SHALL apply a `configuration` entry with the tool it names - `dconf`,
`gsettings`, `macos_defaults`, or `windows_registry` - and SHALL stop the run
with an error naming any other tool. A failure applying a `dconf`,
`macos_defaults`, or `windows_registry` entry SHALL also stop the run.

The only supported `action` is `set`. RWR SHALL record any other declared
action as a ledger failure and skip the entry. The field used to be decoded and
never read, so `action: banana` applied the setting anyway.

For `dconf`, RWR SHALL resolve the entry's `file` relative to the blueprint
directory and feed its content to `dconf load /` on standard input - commands
run without a shell, so a `<` in argv is data, not a redirection. An entry with
`run_once: true` SHALL be skipped when its bootstrap marker file exists, and
the marker SHALL be written only after a successful apply.

For `gsettings`, RWR SHALL check that each key is writable before setting it,
and SHALL record a per-key ledger failure - and continue with the remaining
keys - when the check or the set fails. Every failure here used to be
discarded, so a run in which no setting applied still reported success.

For `macos_defaults`, RWR SHALL write through `defaults write`, defaulting the
domain to `NSGlobalDomain` when the entry declares none.

In dry-run mode RWR SHALL report each configuration it would apply and run
nothing.

#### Scenario: An unsupported action

- **WHEN** a configuration entry declares `action: unset`
- **THEN** the entry is recorded as a ledger failure and skipped
- **AND** the remaining configurations still apply

#### Scenario: A dconf entry marked run-once

- **WHEN** a dconf entry with `run_once: true` runs and its bootstrap marker exists
- **THEN** the entry is skipped without touching dconf

#### Scenario: A read-only gsettings key

- **WHEN** one key in a gsettings entry is not writable
- **THEN** that key is recorded as a ledger failure
- **AND** the remaining keys are still applied

### Requirement: Registry values are written as data, never parsed as code

RWR SHALL write a `windows_registry` entry through PowerShell script bodies
that are compile-time constants, and SHALL pass the blueprint-supplied path,
name, and value in forms PowerShell never re-tokenizes: environment variables
for an unelevated write, and a JSON payload file read with `ConvertFrom-Json`
for an elevated one - a UAC-elevated child does not reliably inherit the
parent's environment. The elevated command itself is a constant plus a
base64-encoded script, whose alphabet contains no quote, space, or
metacharacter.

Supported value types are `string`, `expandstring`, `binary`, `dword`, and
`qword`; any other type SHALL stop the run. Numeric and binary values SHALL be
validated before the write, so a malformed value fails with a named error
instead of reaching the registry as formatted garbage.

Values used to be interpolated into the `-Command` string, so a value such as
`a'; Remove-Item C:\ -Recurse; #` closed the surrounding quote and ran as a
second statement - as administrator, for an elevated entry.

#### Scenario: A value carrying PowerShell metacharacters

- **WHEN** a registry entry's value contains quotes and semicolons
- **THEN** the value is written to the registry verbatim
- **AND** no part of it executes as PowerShell

#### Scenario: A dword given a non-integer

- **WHEN** a `dword` entry's value is not an integer
- **THEN** the run stops with an error naming the value
- **AND** nothing is written to the registry

### Requirement: Fonts install from the latest Nerd Fonts release

RWR SHALL resolve the latest `ryanoasis/nerd-fonts` release once per run and
download each font as `<name>.tar.xz` from that release. The lookup SHALL
happen only after the dry-run and empty-blueprint exits - it is a network call,
and `--dry-run` is expected to work offline.

A failed release lookup SHALL be recorded as a ledger failure and SHALL NOT
abort the run; fonts are cosmetic, and the failure reaches the exit code
through the ledger. One font failing SHALL NOT stop the rest: each failure is
ledgered and processing continues.

A font name SHALL be refused when it is empty or contains a path separator or
`..` - the name is concatenated into the download URL and the local font path,
so `../../owner/repo/x` would both redirect the download to another release and,
on removal, glob outside the font directory.

A download answering anything but HTTP 200 SHALL fail, rather than the error
page being written out as an archive and surfacing later as an unintelligible
decompression error.

#### Scenario: GitHub unreachable

- **WHEN** the release lookup fails
- **THEN** the failure is recorded in the ledger
- **AND** the rest of the run continues

#### Scenario: A traversal in a font name

- **WHEN** a font entry names `../../evil/repo/x`
- **THEN** the entry is refused before any download

### Requirement: Font archives are extracted defensively

RWR SHALL extract only regular-file entries whose name ends in `.ttf` or `.otf`
(case-insensitive) from a font archive, into the system font directory for
`location: system` and the user font directory otherwise. The filter used to be
`.ttf` alone, so an OTF-only archive - several Nerd Fonts ship only `.otf` -
"installed successfully" with zero files written.

Symlink and hardlink entries SHALL be skipped with a warning: links are never
needed to install a font and are the cheapest way to make a later write land
outside the destination. An entry whose path resolves outside the destination
directory SHALL fail the extraction. A single entry decompressing past 64 MB
SHALL fail as a suspected decompression bomb - archives arrive xz-compressed
from the network, and real font faces are single-digit MB.

Each extracted font face SHALL be staged with mode `0644`, copied to its final
destination, and have its staging file removed immediately after that copy.
Failed copies SHALL still clean up their staging files before the run returns.

An archive that produced zero font faces SHALL be a failure, not a success.
After an install or removal RWR SHALL refresh the font cache, elevated only for
a system-scoped install.

Removal SHALL glob both `<name>*.ttf` and `<name>*.otf` in the font directory -
removal was `.ttf`-blind too, so an installed OTF face survived its own
removal.

#### Scenario: An OTF-only archive

- **WHEN** a font archive contains only `.otf` faces
- **THEN** they are installed and counted

#### Scenario: An archive with a traversal entry

- **WHEN** an archive entry names `../../../etc/cron.d/x`
- **THEN** the extraction fails
- **AND** nothing is written outside the font directory

#### Scenario: An archive that installs nothing

- **WHEN** extraction writes zero font faces
- **THEN** the font is recorded as a failure, not reported installed

#### Scenario: Removing a font

- **WHEN** a font entry declares `action: remove`
- **THEN** both `.ttf` and `.otf` faces matching the name are removed
- **AND** the font cache is refreshed

### Requirement: An entry may override interactive mode

RWR SHALL resolve whether an operation prompts by taking the entry's own
`interactive` field when it is set, and the global `--interactive` flag when it
is not. Packages, repositories, directories (in `files`), services, users,
scripts, and `ssh_keys` entries read the field.

This lets a mostly-interactive run declare a known-noisy entry non-interactive,
and the reverse.

#### Scenario: An entry that opts out

- **WHEN** `--interactive` is on and an entry declares `interactive: false`
- **THEN** that entry runs without prompting
- **AND** other entries still prompt

#### Scenario: An entry with no setting

- **WHEN** an entry does not set `interactive`
- **THEN** the global flag decides

### Requirement: Interactive directory copies prompt before overwriting

RWR SHALL show a diff and ask before overwriting when a directory copy runs
interactively and a file already exists at the target. Declining SHALL skip
that file and continue with the rest of the copy.

A failed read of the confirmation - for example EOF on a piped stdin - SHALL
fail the operation with an error naming `--interactive=false` as the way to
skip prompts, and SHALL NOT terminate the process. It used to be a
`log.Fatalf`, which bypassed the failure ledger and every deferred cleanup;
interactive defaults to on, so a piped stdin killed the whole run mid-flight.

#### Scenario: Declining an overwrite

- **WHEN** an interactive copy finds an existing target file and the operator answers `n`
- **THEN** the file is skipped
- **AND** the copy continues with the remaining files

#### Scenario: A prompt with no terminal

- **WHEN** the confirmation read fails
- **THEN** the operation fails with an error suggesting `--interactive=false`
- **AND** the process is not terminated

## Known Gaps

- **The users and scripts processors do not use the failure ledger.** The other
  eight processors - packages, repositories, services, files (with its templates
  and directories sections), configuration, git, `ssh_keys`, and fonts - record a
  failed item and keep going. The users and scripts processors return an error on
  the first failed entry, which aborts the rest of the run: the failure does reach
  the exit code, but the remaining work is given up rather than attempted and
  reported at the end.
- **The files processor's `target` is not contained.** Repository steps resolve
  their paths against the provider's declared repository directories - the editing
  and removing steps and, since the write-path containment landed, the `download`,
  `write` and `copy` steps too. The files processor still writes to whatever
  `target` a blueprint names, with no boundary check.
- **Windows users are unimplemented.** `ProcessUsers` has Linux (shadow-utils) and
  macOS (Open Directory) implementations; on Windows it logs a warning per entry
  and does nothing.
