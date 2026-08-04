# Provider Detection Specification

## Purpose

A provider is a declarative description of a package manager: the binary that
identifies it, the files that prove it is in use, the distributions it belongs to,
and the command templates for install, remove, update, list, search, and clean.
Providers are what let RWR put "the annoying bits on rails" - a blueprint says
`action: install`, and the provider decides what that means on this machine.

This capability defines how providers are loaded, how RWR decides which ones are
usable, and how it picks a default.
## Requirements
### Requirement: Providers ship embedded and can be overridden from disk

RWR SHALL embed its provider definitions in the binary, so a fresh machine with no
files on it can still install packages.

RWR SHALL also load provider definitions from a filesystem directory when one is
found. A filesystem provider SHALL replace an embedded provider of the same name.

RWR SHALL fail only when it has no providers at all.

#### Scenario: A machine with no provider directory

- **WHEN** RWR starts on a machine with no `providers/` directory anywhere
- **THEN** the embedded providers load and the run proceeds

#### Scenario: An operator overriding a provider

- **WHEN** a `providers/` directory contains a definition named `pacman`
- **THEN** that definition replaces the embedded `pacman`

### Requirement: A provider is only loaded from a directory only its owner can write

RWR SHALL search for a filesystem provider directory in the executable's own
directory, `/usr/local/share/rwr/providers`, `/usr/share/rwr/providers`, and
`~/.config/rwr/providers` - plus the Homebrew and app-bundle paths on macOS.

RWR SHALL NOT search the current working directory.

RWR SHALL skip a provider file that is group- or world-writable, warning which file
and which mode, and SHALL continue loading the rest of the directory.

A provider file declares `exec`, `args` and `elevated = true`, and those are run
verbatim. Honouring `./providers` handed root-level execution to any directory RWR
happened to be run from - a cloned blueprint repo, `/tmp`, a shared downloads
folder - and a definition anyone on the box can edit is a root shell for anyone on
the box. One bad file is skipped rather than failing the load, so it does not cost
the operator every other provider in the directory.

#### Scenario: Running from a directory containing providers

- **WHEN** RWR is run from a directory that contains a `providers/` subdirectory
- **THEN** those definitions are not loaded

#### Scenario: A world-writable provider file

- **WHEN** `~/.config/rwr/providers/paru.toml` is mode `0666`
- **THEN** RWR warns and skips that file
- **AND** the other definitions in the directory still load

### Requirement: Provider availability is decided by evidence on the machine

RWR SHALL treat a provider as available when all of these hold:

1. The provider targets this operating system.
2. The provider's declared binary is on `PATH`.
3. Every file the provider declares as detection evidence exists.

When those hold and the provider's distribution list also names this distribution
or its family, the provider SHALL be available.

When those hold but the distribution is not named, RWR SHALL still treat the
provider as available if it declares detection files - because a machine that has
`pacman` on `PATH`, `/etc/pacman.conf`, and `/var/lib/pacman` is running `pacman`
whatever `/etc/os-release` calls itself.

A provider that declares no detection files SHALL have only its distribution list to
go on, and SHALL be unavailable on an unrecognised distribution.

This exists because derivatives outnumber any list that can be maintained. It was
found on PrismLinux, which reports `ID=prismlinux`, sets no `ID_LIKE`, and is named
by no provider - yet is plainly an Arch system.

#### Scenario: An unrecognised Arch derivative

- **WHEN** `/etc/os-release` reports a distribution no provider names
- **AND** `pacman` is on `PATH` with `/etc/pacman.conf` and `/var/lib/pacman` present
- **THEN** the `pacman` provider is available

#### Scenario: A distribution that lies about itself

- **WHEN** a machine calls itself Arch but installs packages with `apt`
- **AND** neither the `apt` binary nor `/etc/apt` is present
- **THEN** the `apt` provider is unavailable

#### Scenario: A Windows provider on Linux

- **WHEN** a provider names `windows` in its distributions and the host is Linux
- **THEN** the provider is unavailable regardless of any other evidence

### Requirement: Distribution families are resolved without enumerating derivatives

RWR SHALL map a distribution to its family, so a provider that names `arch` matches
EndeavourOS, Manjaro, Garuda and the rest without naming each one.

RWR SHALL consult `/etc/os-release` `ID_LIKE` only when the distribution being asked
about is the host's own. `ID_LIKE` describes the host and nothing else; consulting
it for an arbitrary distribution answers every question in terms of whatever the
host happens to be, which silently maps unrecognised distributions onto the host's
family.

#### Scenario: A known derivative

- **WHEN** the family of `endeavouros` is requested
- **THEN** the answer is `arch`

#### Scenario: ID_LIKE scoped to the host

- **WHEN** the host reports `ID_LIKE=debian`
- **AND** RWR is asked whether `arch` is in the `debian` family
- **THEN** the answer is no

### Requirement: Every available provider is recorded as a package manager

RWR SHALL record each available provider in the detected package-manager map, with
its resolved binary path and its command templates.

RWR SHALL NOT re-filter that map by matching the provider's distribution list
against a literal OS name. Doing so excluded every provider that names concrete
distributions - `apt`, `dnf`, `pacman`, `zypper`, and the AUR helpers - leaving
cache cleaning and core-package resolution operating on a map that held only
wildcard providers.

#### Scenario: An Arch machine with paru installed

- **WHEN** provider detection runs on a machine with `pacman` and `paru` available
- **THEN** both appear in the package-manager map with usable binary paths
- **AND** cache cleaning issues a `pacman` clean command

### Requirement: A provider that declares no clean command is not invoked

At the end of a run RWR SHALL issue a clean command only for providers that declare
one, deciding by the clean arguments rather than by the assembled command string.

A package manager's `Clean` is built as `"<bin> <clean args>"`, so a provider with
no clean command still yields `"<bin> "` and never the empty string. Testing the
concatenation therefore never skipped anything, and the bare provider binary was
executed at the end of every run. For an AUR helper, a bare invocation can mean a
full system upgrade.

#### Scenario: A provider with no clean command

- **WHEN** an available provider's definition declares no clean command
- **THEN** nothing is spawned for it during cleanup
- **AND** its bare binary is not invoked

#### Scenario: A provider with a clean command

- **WHEN** a provider declares a clean command
- **THEN** it is spawned with the provider's binary and the declared arguments

### Requirement: The default package manager is deterministic

RWR SHALL choose the default package manager as the first entry of the platform's
preference list that is present. When no preferred manager is present, RWR SHALL
fall back to the alphabetically first available manager.

RWR SHALL NOT choose by map iteration order. Go randomizes map iteration, so a
package with no explicit `package_manager` could otherwise be installed by a
different tool on each run.

#### Scenario: Repeated runs on the same machine

- **WHEN** default selection runs many times on an unchanged machine
- **THEN** it resolves to the same manager every time

#### Scenario: A machine with only unpreferred managers

- **WHEN** no manager from the preference list is installed
- **AND** `flatpak` and `snap` are both available
- **THEN** the default is `flatpak`, chosen by sorted name rather than at random

### Requirement: Embedded provider definitions are CUE, and only CUE

Embedded provider definitions SHALL be authored in CUE under
`internal/system/definitions/`, embedded in the binary, and evaluated at
load; there SHALL be no second committed representation. The schema closes
`#Provider` and unifies every entry against it, so an invalid definition
fails evaluation naming the field.

Why: the export-and-commit pipeline made every provider change a
multi-format edit - the disease the CUE migration existed to cure.

#### Scenario: Provider missing a required field fails at load

- **GIVEN** a CUE provider definition lacking `commands.install`
- **WHEN** the embedded definitions are evaluated
- **THEN** evaluation fails naming the provider and field

### Requirement: Filesystem overrides do not require CUE

Filesystem provider overrides SHALL be accepted as `.toml` or `.json`.
Operators SHALL NOT need a CUE toolchain to override a provider.

#### Scenario: JSON override

- **GIVEN** `~/.config/rwr/providers/pacman.json` holding a full provider document
- **WHEN** providers initialize
- **THEN** the override replaces the embedded pacman definition, same as a
  TOML override does today

### Requirement: CUE sources are vetted in CI

CI SHALL run `cue vet` over the embedded CUE sources: a schema violation
fails with CUE's own error naming the field.

#### Scenario: Schema violation fails CI

- **GIVEN** a PR gives a provider step an action outside the enum
- **WHEN** CI runs
- **THEN** the `cue-providers` job fails naming the field

