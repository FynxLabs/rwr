# CLI Specification

## Purpose

The command surface an operator actually touches. This capability defines the
commands, the global flags, and which commands need a working configuration.
## Requirements
### Requirement: `rwr all` applies the whole tree

`rwr all` SHALL run every blueprint type in the resolved run order. This is the
new-machine command.

#### Scenario: Setting up a new machine

- **WHEN** `rwr all` runs with a valid init file
- **THEN** package managers, then bootstrap, then each blueprint type run in order

### Requirement: `rwr run <processor>` applies one blueprint type

`rwr run` SHALL provide a subcommand for each blueprint type — `packages`,
`repository`, `services`, `files`, `configuration`, `users`, `git`, `scripts`,
`ssh_keys`, and `fonts` — plus `all`, which runs everything exactly as `rwr all`
does. The subcommands are generated from one processor table, so a new processor
is one table entry, not a new hand-written command.

Each processor subcommand SHALL apply only that type, using the same processing
path as `rwr all`.

A bare `rwr run` SHALL list the processors and exit zero, like a task runner
listing its tasks; naming an unknown processor SHALL print the list and fail.

Each processor name SHALL also work directly off the root — `rwr packages` is
`rwr run packages` — without the processor names entering the primary command
namespace, so they cannot collide with real subcommands.

#### Scenario: Reapplying packages only

- **WHEN** `rwr run packages` runs
- **THEN** only the packages blueprints are processed

#### Scenario: Listing the processors

- **WHEN** `rwr run` runs with no processor named
- **THEN** the processor list is printed and the exit code is zero

#### Scenario: The root shorthand

- **WHEN** `rwr packages` runs
- **THEN** it behaves exactly as `rwr run packages`

### Requirement: `rwr validate` checks configuration without applying it

`rwr validate [path]` SHALL check blueprints or providers and SHALL make no change to
the machine. It SHALL default to the current directory.

`rwr validate` SHALL NOT require the init file to load successfully.

#### Scenario: Checking before applying

- **WHEN** `rwr validate ./blueprints` runs
- **THEN** issues are reported and no package, file, or service is touched

### Requirement: `rwr config --create` writes the configuration file

`rwr config --create` SHALL write RWR's own configuration file. Without the flag it
SHALL show help.

The written file SHALL be owner-readable only, because it holds credentials.

#### Scenario: First-time setup

- **WHEN** `rwr config --create` runs
- **THEN** `~/.config/rwr/config.yaml` is written at `0600`

### Requirement: `rwr profiles` lists the profiles a tree defines

`rwr profiles` SHALL read the blueprint tree and report every profile its entries
declare, with the number of entries carrying each and the number of base items that
always apply.

Reading only the init file's inline arrays reports nothing for a tree that declares
profiles on blueprint entries, which is where they are normally declared.

The usage examples it prints SHALL be commands that exist.

#### Scenario: Discovering profiles

- **WHEN** `rwr profiles` runs against a tree whose blueprints declare `work` and
  `personal`
- **THEN** both names are listed with their entry counts

#### Scenario: A tree with no profiles

- **WHEN** no blueprint entry declares a profile
- **THEN** RWR reports that all items are base items and always apply

### Requirement: Global flags apply to every command

RWR SHALL accept these flags on any command:

| Flag | Effect |
|---|---|
| `--init-file`, `-i` | Path, directory, or URL of the init file |
| `--profile`, `-p` | Activate a profile; may be repeated |
| `--dry-run` / `--no-op` | Log operations without performing them |
| `--debug`, `-d` | Debug logging |
| `--log-level` | `debug`, `info`, `warn`, or `error` |
| `--interactive`, `-I` | Interactive mode; on by default |
| `--force-bootstrap` | Run bootstrap again |
| `--show-secrets` | Print credential values instead of redacting |
| `--gh-api-key` / `--gh-key` | GitHub API token |
| `--gh-auth` | Authenticate with GitHub by OAuth device flow |
| `--ssh-key` | SSH private key path or base64 value |
| `--skip-version-check` | Do not check for a newer RWR |
| `--config` | Config file, or a directory containing `config.yaml` |
| `--version`, on the root command | Print the version and exit |

#### Scenario: A non-interactive run

- **WHEN** `rwr all --interactive=false` runs and a choice is needed
- **THEN** RWR takes the documented default rather than prompting

### Requirement: `rwr version` reports what this binary is

RWR SHALL provide a `version` subcommand and a `--version` flag on the root command,
both reporting the version injected at link time by goreleaser, along with the
commit, build date, builder, tree state, and the Go toolchain and platform.

When a field was not injected — which is the case for `go build` and `go install`
binaries — RWR SHALL fall back to the module and VCS information the Go toolchain
embeds, and SHALL report `dev` rather than nothing.

A binary on disk that cannot say which commit produced it cannot be matched to a
bug report.

#### Scenario: A release binary

- **WHEN** `rwr version` runs on a released build
- **THEN** the version, commit, build date and builder are printed

#### Scenario: A locally built binary

- **WHEN** `rwr version` runs on a plain `go build` binary
- **THEN** the version reads `dev` and the commit comes from the embedded VCS data

### Requirement: `--skip-version-check` suppresses a real check

At the start of any command that initializes the system, RWR SHALL ask GitHub for
the latest release and SHALL print a one-line notice to stderr when a newer version
exists. `--skip-version-check` SHALL suppress it, as SHALL the `rwr.skipVersionCheck`
configuration key and its environment form — the flag binding into viper is one-way,
so reading only the flag variable left the key `rwr config --create` writes inert.

The check SHALL be strictly advisory: a timeout, an unreachable network, a rate
limit, or a response RWR cannot parse SHALL each be a debug log and a silent
return, never a failed run.

RWR SHALL skip the check for a dev build, so a plain `go build` in a checkout never
reaches out to the network.

The flag previously existed and guarded nothing.

#### Scenario: A newer release exists

- **WHEN** a released binary older than the latest release runs `rwr all`
- **THEN** a one-line notice naming the newer version is printed to stderr and the
  run proceeds

#### Scenario: GitHub is unreachable

- **WHEN** the release lookup fails
- **THEN** the run proceeds with no notice and no error

#### Scenario: The check is suppressed

- **WHEN** `--skip-version-check` is passed, or `rwr.skipVersionCheck` is set in the
  config file
- **THEN** no request is made

### Requirement: A runtime failure does not print the usage text

RWR SHALL NOT print command usage or the flag listing when a command fails during
execution, and SHALL report each error exactly once.

A run that fails partway through is not a usage mistake. Printing the full flag
listing after "validation failed with 3 errors" buries the errors the operator
needs to read under a screen of help text, and cobra printing its own error line
alongside RWR's produced every failure twice.

#### Scenario: A failing validate run

- **WHEN** `rwr validate` finds errors
- **THEN** the errors are printed once and no usage text follows

### Requirement: `remove` and `delete` mean the same thing for users and groups

The users processor SHALL accept `remove` as the canonical action and `delete` as an
alias, for both users and groups. `rwr validate` SHALL accept both.

Previously each name failed at the opposite end of the run: the processor took only
one of them and the validator only the other, so whichever the operator wrote was
rejected somewhere.

#### Scenario: A blueprint using the alias

- **WHEN** a users blueprint declares `action: delete`
- **THEN** validation reports no error
- **AND** the run removes the account

### Requirement: Commands that do not act on the system skip initialization

RWR SHALL skip loading the init file, cloning the blueprint repository, and detecting
package managers for `help`, `version`, `config`, and `validate`.

`validate` SHALL still detect the operating system and set paths, because validation
reports on the current platform.

#### Scenario: Getting help with no configuration

- **WHEN** `rwr help` runs on a machine with no init file
- **THEN** help is shown without error

### Requirement: --config-name selects a manifest entry explicitly

`--config-name <name>` SHALL select the named manifest entry, bypassing
matching and prompting, and SHALL error if the name does not exist. In a
non-TTY run with multiple matches and no `--config-name`, the system SHALL
error listing the candidates rather than prompt.

Why: scripts and CI must never block on a prompt.

#### Scenario: Non-TTY with multiple matches

- **GIVEN** a manifest with two matching entries and stdout not a terminal
- **WHEN** `rwr all` runs without `--config-name`
- **THEN** the run exits nonzero listing `arch-desktop` and `arch-server`

#### Scenario: Explicit selection wins

- **GIVEN** `--config-name arch-server` on a host that would auto-match
  `arch-desktop`
- **WHEN** `rwr all` runs
- **THEN** `arch-server` is used

## Known Gaps

- **`--gh-api-key` and `--gh-key` bind the same configuration key.** Passing both
  silently lets one win rather than reporting the conflict.
- **The version notice compares only dotted numeric versions.** A version string it
  cannot parse compares as equal, so no notice is shown rather than a wrong one.
