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

RWR SHALL refuse an `http://` URL. The init file decides which repository the
blueprints come from, which package managers are installed, and which scripts run
elevated; served in cleartext it can be rewritten by anyone on the path.

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

#### Scenario: An init file over plain http

- **WHEN** the init file is given as an `http://` URL
- **THEN** RWR refuses with an error naming the URL and fetches nothing

### Requirement: The init file is rendered as a template before it is read

RWR SHALL resolve template variables in the init file before parsing it, with the
same `User`, `System`, `Flags`, and `UserDefined` variables available to blueprints.

#### Scenario: A blueprint location under the user's home

- **WHEN** the init file declares `location: {{ .User.home }}/blueprints`
- **THEN** the location resolves to the running user's home directory

### Requirement: Variables declared in the init file reach blueprint templates

RWR SHALL decode the init file's `variables.userDefined` block and SHALL make its
keys available to every blueprint template as `{{ .UserDefined.<key> }}`.

The runtime halves of the variable set — `Flags`, `User`, `System` — are computed
rather than declared and SHALL NOT be decodable from the init file, so a blueprint
cannot claim to be running as a different user or with different flags than it is.
Filling them in SHALL preserve the declared `userDefined` entries rather than
replacing the whole structure.

Previously the block decoded into nothing and was then overwritten, so every
`{{ .UserDefined.x }}` in a tree rendered empty — and because rendering is strict,
the run stopped on a variable the operator had in fact declared.

`RWR_`-prefixed environment variables SHALL be merged into the same namespace with
the prefix removed, so the operator can supply a value at run time that the init
file does not carry.

#### Scenario: A value declared in the init file

- **WHEN** the init file declares `variables.userDefined.company: acme`
- **AND** a blueprint references `{{ .UserDefined.company }}`
- **THEN** it renders as `acme`

#### Scenario: A value supplied by the environment

- **WHEN** `RWR_company=acme` is exported
- **THEN** `{{ .UserDefined.company }}` renders as `acme`

### Requirement: Templates are always processed

RWR SHALL process the `templates` section of a files blueprint unconditionally.
There SHALL be no `blueprints.templatesEnabled` option.

A switch that turned templating off left the `templates` section of every files
blueprint silently unprocessed, which is not a state anybody wants a machine in.

#### Scenario: A tree declaring templates

- **WHEN** a files blueprint declares a `templates` section
- **THEN** the templates are rendered and written, with no option required to
  enable them

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

RWR SHALL accept a `--config` flag naming either a config file or a directory
containing `config.yaml`. When a file is named, its directory SHALL become the
config location, so the run-once state directory stays next to the config it
belongs to.

Configuration keys are nested (`log.level`), and a dot is not legal in an
environment variable name, so RWR SHALL translate dots to underscores when reading
the environment — `RWR_LOG_LEVEL` sets `log.level`.

RWR SHALL create its config directory and its run-once state directory at startup,
at owner-only permissions.

#### Scenario: A token supplied by flag

- **WHEN** `--gh-api-key` is given and a token is also in the config file
- **THEN** the flag value is used

#### Scenario: A config directory given on the command line

- **WHEN** `--config /srv/rwr` is given and that path is a directory
- **THEN** `config.yaml` is read from it and `run_once` is created inside it

#### Scenario: A log level from the environment

- **WHEN** `RWR_LOG_LEVEL=debug` is exported
- **THEN** the run logs at debug level

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
- **Config keys containing a hyphen cannot be set from the environment.** The dot-
  to-underscore replacer makes `RWR_LOG_LEVEL` reach `log.level`, but `rwr.init-file`
  would need `RWR_RWR_INIT-FILE`, and a hyphen is not legal in an environment
  variable name. That key is only reachable by flag or config file.
- **Init source resolution is not centralized.** The forms above are handled in two
  places rather than by one resolver, which is what makes adding shorthands awkward.
- **The installer does not take an init path.** A `curl | bash` one-liner cannot yet
  pass the init file through to a first run, which is what would make new-machine
  setup a single command.
- **Managed GitHub auth and SSH bootstrap.** Detecting missing GitHub auth during
  init, running an interactive device-flow login, generating or importing a key,
  registering it, and verifying the handshake is wanted and unbuilt. The OAuth
  device flow exists behind `--gh-auth`; the rest does not.
