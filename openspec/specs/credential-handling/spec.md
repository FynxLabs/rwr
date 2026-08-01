# Credential Handling Specification

## Purpose

RWR holds two credentials: a GitHub API token and an SSH private key. Blueprints are
cloned from git repositories, and everything a blueprint can read, the author of that
blueprint can read — a template writes its result to a path the same blueprint
chooses, and a script inherits the environment RWR hands it. RWR cannot tell a
blueprint the operator wrote from one they pulled in.

This capability defines where credentials may go, how an operator opts a blueprint
tree into the ones it genuinely needs, and how values are kept out of logs.

## Requirements

### Requirement: Credentials are withheld from blueprints by default

RWR SHALL NOT place the GitHub API token or the SSH private key into blueprint
template scope, and SHALL NOT export them into the environment of any command it
spawns, unless the init file opted into them by name.

Every other configuration key SHALL continue to be exported as `RWR_VAR_*`.

#### Scenario: A default tree running a script

- **WHEN** an init file names no exposed credentials and a blueprint runs a script
- **THEN** `RWR_VAR_REPOSITORY_GH_API_TOKEN` and `RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY`
  are absent from the script's environment

#### Scenario: A default tree rendering a template

- **WHEN** a template references `{{ .Flags.ghAPIToken }}` and no opt-in was given
- **THEN** the value does not appear in the rendered output

### Requirement: A tree opts into the credentials it needs, by name

RWR SHALL accept an `exposeCredentials` list in the init file naming the credentials
that this tree's blueprints may read. RWR SHALL share only the named credentials.

RWR SHALL accept both the bare name (`gh_api_token`) and the full configuration key
(`repository.gh_api_token`) as the same credential.

RWR SHALL warn at the start of the run when any credential is exposed, so the change
is visible rather than silent.

This is opt-in rather than unavailable because some blueprints legitimately need a
token — writing a `.netrc`, configuring `gh`, calling the GitHub API from a script.

#### Scenario: Opting into the token only

- **WHEN** an init file declares `exposeCredentials: [gh_api_token]`
- **THEN** `{{ .Flags.ghAPIToken }}` renders and `RWR_VAR_REPOSITORY_GH_API_TOKEN`
  is present in spawned commands
- **AND** the SSH private key remains withheld

#### Scenario: Exposure is announced

- **WHEN** a run starts with any credential exposed
- **THEN** RWR logs a warning naming the exposed credentials

### Requirement: Credential values are redacted in logs

RWR SHALL substitute a redaction placeholder wherever a credential value would
otherwise be printed, including when a value is formatted as part of a larger
structure.

Redaction SHALL apply to exposed credentials as well as withheld ones.

#### Scenario: Debug logging with a token configured

- **WHEN** debug logging is on and a GitHub token is configured
- **THEN** the token value does not appear in the log output

#### Scenario: A flags value formatted into a log line

- **WHEN** the flags structure is printed with `%v` or `%+v`
- **THEN** the credential fields render as the redaction placeholder

### Requirement: Showing credential values requires an explicit flag

RWR SHALL provide a `--show-secrets` flag that prints credential values instead of
redacting them, and SHALL warn while that flag is active.

This exists because "is RWR even reading my token?" has no other answer. It is not
tied to debug logging, so turning debug on does not reveal values as a side effect.

#### Scenario: Confirming a token is being read

- **WHEN** `--show-secrets` is passed
- **THEN** credential values appear in the log
- **AND** RWR warns that values will appear in logs

### Requirement: RWR's own config tree is owner-only

RWR SHALL create its config and state directories at `0700` and SHALL restrict its
written config file to `0600`, because that file holds the GitHub token and a
base64-encoded SSH private key.

Tightening SHALL never widen: a directory or file an operator restricted further
SHALL keep their setting.

#### Scenario: A fresh config directory

- **WHEN** RWR creates `~/.config/rwr` for the first time
- **THEN** the directory mode is `0700`

#### Scenario: An existing permissive directory

- **WHEN** `~/.config/rwr` exists at `0777`
- **THEN** RWR tightens it to `0700`

#### Scenario: An operator who restricted further

- **WHEN** `~/.config/rwr` exists at `0500`
- **THEN** RWR leaves it at `0500`

### Requirement: RWR does the credentialed work itself where it can

Where RWR can perform a credentialed operation on the blueprint's behalf, it SHALL
do so rather than handing the credential over. Registering a generated SSH key with
GitHub SHALL use the token from RWR's own configuration; the blueprint SHALL NOT
need to read it.

#### Scenario: Publishing a generated key

- **WHEN** an `ssh_keys` blueprint asks for a key to be added to GitHub
- **THEN** RWR calls the GitHub API with the configured token
- **AND** the blueprint never receives the token
