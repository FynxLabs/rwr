# Initialization Specification

## Purpose

The init file is the entry point: it says where the blueprints are, what format they
are in, what order to run them, which package managers to install first, and which
credentials the tree may read. Making a new machine a one-shot operation means making
that file easy to point at from anywhere.

## Requirements

### Requirement: The init file can be a path, a directory, or a URL

RWR SHALL accept an init file given as:

- a path to a file
- a path to a directory, in which case RWR SHALL look for `init.yaml`, `init.yml`,
  `init.json`, then `init.toml` inside it
- an `https://` URL to a raw file
- a GitHub blob URL, which RWR SHALL rewrite to its raw form

When no init file is given, RWR SHALL look for `init.yaml`, `init.yml`, `init.json`,
then `init.toml` in the current directory.

The init file location MAY also be set in RWR's own configuration.

#### Scenario: Pointing at a directory

- **WHEN** `--init-file ./blueprints` is given and that directory holds `init.yaml`
- **THEN** that file is used

#### Scenario: A GitHub blob URL

- **WHEN** the init file is given as a `github.com/.../blob/...` URL
- **THEN** RWR downloads the corresponding `raw.githubusercontent.com` content

#### Scenario: No init file given

- **WHEN** `rwr all` runs in a directory containing `init.yaml`
- **THEN** that file is used

### Requirement: The init file is rendered as a template before it is read

RWR SHALL resolve template variables in the init file before parsing it, with the
same `User`, `System`, `Flags`, and `UserDefined` variables available to blueprints.

#### Scenario: A blueprint location under the user's home

- **WHEN** the init file declares `location: {{ .User.home }}/blueprints`
- **THEN** the location resolves to the running user's home directory

### Requirement: The blueprint location resolves against the init file

RWR SHALL resolve the blueprint location as follows:

- empty or `.` — the directory containing the init file
- beginning with `~` — relative to the user's home directory
- a relative path — relative to the directory containing the init file
- an absolute path — used as given

#### Scenario: A relative location

- **WHEN** an init file at `/srv/cfg/init.yaml` declares `location: blueprints`
- **THEN** the blueprint location is `/srv/cfg/blueprints`

### Requirement: Paths are expanded by RWR, not by a shell

RWR SHALL expand a leading `~` in every blueprint-supplied path before that path
reaches a command or a filesystem call.

Commands are built as argv with no shell interposed, so nothing else performs this
expansion. Without it, a blueprint declaring `path: ~/.ssh` creates a directory
literally named `~` in the working directory.

#### Scenario: An SSH key path

- **WHEN** an `ssh_keys` blueprint declares `path: ~/.ssh`
- **THEN** `ssh-keygen` receives the absolute path under the user's home directory
- **AND** no directory named `~` is created

#### Scenario: A git clone target

- **WHEN** a git blueprint declares a target beginning with `~`
- **THEN** the repository is cloned under the user's home directory

### Requirement: Configuration comes from flags, the config file, and the environment

RWR SHALL read its own configuration from `~/.config/rwr/config.yaml`, and SHALL
allow any key to be set from the environment with an `RWR_` prefix. Command-line
flags SHALL take precedence.

RWR SHALL create its config directory and its run-once state directory at startup,
at owner-only permissions.

#### Scenario: A token supplied by flag

- **WHEN** `--gh-api-key` is given and a token is also in the config file
- **THEN** the flag value is used

### Requirement: TOML init files are converted before parsing

RWR SHALL convert a TOML init file to YAML before reading it, because the
configuration reader does not handle the TOML directly.

#### Scenario: A TOML init file

- **WHEN** the init file is `init.toml`
- **THEN** it is parsed correctly and produces the same configuration as the
  equivalent YAML

## Known Gaps

These are wanted and not yet implemented.

- **GitHub repository shorthands.** `owner/repo`, `owner/repo@ref`, and
  `owner/repo/path/to/init.yaml` are not accepted. Only a full URL, a local path, or
  a directory works.
- **Init source resolution is not centralized.** The forms above are handled in two
  places rather than by one resolver, which is what makes adding shorthands awkward.
- **`http://` is accepted.** A plaintext init file is downloaded without complaint.
  It should require an explicit opt-in flag.
- **The installer does not take an init path.** A `curl | bash` one-liner cannot yet
  pass the init file through to a first run, which is what would make new-machine
  setup a single command.
- **Managed GitHub auth and SSH bootstrap.** Detecting missing GitHub auth during
  init, running an interactive device-flow login, generating or importing a key,
  registering it, and verifying the handshake is wanted and unbuilt. The OAuth
  device flow exists behind `--gh-auth`; the rest does not.
