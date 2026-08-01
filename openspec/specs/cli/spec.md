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

### Requirement: `rwr run <type>` applies one blueprint type

`rwr run` SHALL provide a subcommand for each blueprint type: `packages`,
`repository`, `services`, `files`, `configuration`, `users`, `git`, `scripts`,
`ssh_keys`, and `fonts`.

Each SHALL apply only that type, using the same processing path as `rwr all`.

#### Scenario: Reapplying packages only

- **WHEN** `rwr run packages` runs
- **THEN** only the packages blueprints are processed

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

#### Scenario: A non-interactive run

- **WHEN** `rwr all --interactive=false` runs and a choice is needed
- **THEN** RWR takes the documented default rather than prompting

### Requirement: Commands that do not act on the system skip initialization

RWR SHALL skip loading the init file, cloning the blueprint repository, and detecting
package managers for `help`, `version`, `config`, and `validate`.

`validate` SHALL still detect the operating system and set paths, because validation
reports on the current platform.

#### Scenario: Getting help with no configuration

- **WHEN** `rwr help` runs on a machine with no init file
- **THEN** help is shown without error

## Known Gaps

- **`--gh-api-key` and `--gh-key` bind the same configuration key.** Passing both
  silently lets one win rather than reporting the conflict.
- **`rwr run` is twelve near-identical subcommands.** One command taking the type as
  an argument would do the same work.
