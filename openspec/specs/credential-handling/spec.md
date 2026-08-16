# Credential Handling Specification

## Purpose

RWR holds two credentials: a GitHub API token and an SSH private key. Blueprints are
cloned from git repositories, and everything a blueprint can read, the author of that
blueprint can read - a template writes its result to a path the same blueprint
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
token - writing a `.netrc`, configuring `gh`, calling the GitHub API from a script.

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

### Requirement: The GitHub token is sent only to GitHub

RWR SHALL attach the configured GitHub token only to remotes whose host is
`github.com`, `www.github.com`, or `raw.githubusercontent.com`. For any other
`https://` host RWR SHALL clone anonymously and SHALL warn naming the host.

RWR SHALL refuse an `http://` remote for a repository declared `private`, rather
than sending the credential over it, and SHALL refuse any scheme other than
`https://` or `git@` when authentication is requested.

A blueprint's `url` is attacker-controlled input whenever the blueprint is. The
token used to be attached to every non-SSH remote, so
`{url: "http://attacker.tld/r.git", private: true}` mailed the operator's PAT to
the attacker in cleartext.

#### Scenario: A private repository on a third-party host

- **WHEN** a blueprint declares a private repository at `https://git.example.com/x.git`
- **THEN** RWR warns that the token is not being sent and clones without
  authentication

#### Scenario: A private repository over plaintext

- **WHEN** a blueprint declares a private repository at `http://example.com/x.git`
- **THEN** RWR refuses with an error naming the URL and sends no credential

#### Scenario: A GitHub remote

- **WHEN** a private repository at `https://github.com/owner/repo.git` is cloned
- **THEN** the configured token is used as the HTTP credential

### Requirement: The init file is never fetched over plaintext

RWR SHALL refuse an init file given as an `http://` URL.

The init file drives everything RWR then runs - which repository the blueprints
come from, which package managers are installed, which scripts execute elevated.
Over `http://` that document can be rewritten by anyone on the path.

#### Scenario: An init file given over http

- **WHEN** `--init-file http://example.com/init.yaml` is given
- **THEN** RWR refuses with an error naming the URL and downloads nothing

### Requirement: The interactive config creator does not echo stored credentials

RWR SHALL show the current value redacted, rather than printing it as the prompt
default, when `rwr config --create` asks for the GitHub token or the SSH private key.

The prompt's purpose is to say whether a value is already stored, which a redaction
placeholder answers just as well; printing the value writes a token and a private
key onto the operator's terminal and into their scrollback.

#### Scenario: Re-running the config creator

- **WHEN** `rwr config --create` runs on a machine with a token already configured
- **THEN** the prompt shows the redaction placeholder as the current value
- **AND** pressing enter keeps the stored value

### Requirement: GitHub authentication runs by OAuth device flow

RWR SHALL authenticate to GitHub by the OAuth device flow when `--gh-auth` is
given or the operator chooses OAuth at the interactive prompt: request a device
code scoped to `write:public_key`, display the verification URL and user code,
and poll for the token at the server-provided interval (default five seconds),
continuing on `authorization_pending` and backing off when GitHub answers
`slow_down`. Polling SHALL give up after five minutes.

The token obtained SHALL be saved to the OS keyring when a backend is available,
without writing it to a plaintext configuration file. When no keyring backend
is available, RWR SHALL warn and use the owner-only (`0600`) configuration-file
fallback. A failure to save SHALL NOT discard the token: the run continues with
it and tells the operator how to supply it directly next time.

When a different token is already stored and the run is interactive, RWR SHALL
ask before replacing it, and declining SHALL keep the stored token. A manually
entered token SHALL be read without echo and refused unless it carries a
recognized GitHub prefix (`ghp_`, `gho_`, `ghu_`).

In a non-interactive run needing a token none is configured for, RWR SHALL fail
with an error listing the ways to supply one - `--gh-api-key`, `--gh-auth`, or
`GITHUB_TOKEN` - rather than prompting.

#### Scenario: A device-flow login

- **WHEN** `--gh-auth` runs and the operator authorizes the code in a browser
- **THEN** RWR obtains the token, persists it using the keyring-first policy,
  and uses it

#### Scenario: The operator never authorizes

- **WHEN** five minutes pass without authorization
- **THEN** the flow fails with a timeout error

#### Scenario: A token already stored

- **WHEN** a different token already exists in the config and the run is interactive
- **THEN** RWR asks before replacing it
- **AND** declining keeps the stored token

#### Scenario: Non-interactive with no token

- **WHEN** a run with `--interactive=false` needs a GitHub token and none is configured
- **THEN** the run fails listing `--gh-api-key`, `--gh-auth`, and `GITHUB_TOKEN`

### Requirement: RWR does the credentialed work itself where it can

Where RWR can perform a credentialed operation on the blueprint's behalf, it SHALL
do so rather than handing the credential over. Registering a generated SSH key with
GitHub SHALL use the token from RWR's own configuration; the blueprint SHALL NOT
need to read it.

#### Scenario: Publishing a generated key

- **WHEN** an `ssh_keys` blueprint asks for a key to be added to GitHub
- **THEN** RWR calls the GitHub API with the configured token
- **AND** the blueprint never receives the token

### Requirement: Trees declare the credentials they need

RWR SHALL accept a `credentials` section in the init file declaring named
credentials, each with an ordered list of sources (`env:<VAR>`, `keyring`,
`prompt`), and SHALL resolve every declared credential before any processor
runs, taking the first source that yields a value. A declared credential that
resolves from no source SHALL fail the run with an error naming the credential
and the sources tried.

Declared credentials receive the same handling as the built-in GitHub token and
SSH private key: withheld from template scope and spawned-command environments
unless named in `exposeCredentials`, and redacted in logs.

#### Scenario: Declared credential resolves from the environment

- **WHEN** an init file declares `cachix_token` with sources
  `[env:CACHIX_AUTH_TOKEN, prompt]` and `CACHIX_AUTH_TOKEN` is set
- **THEN** the credential resolves without prompting
- **AND** its value is absent from spawned-command environments unless
  `exposeCredentials` names it

#### Scenario: Unresolvable credential fails up front

- **WHEN** a declared credential yields no value from any source in a non-TTY
  run
- **THEN** the run fails before any processor executes, with an error naming
  the credential and noting that `prompt` was skipped

### Requirement: Managed credentials are never stored in plaintext

RWR SHALL NOT write a managed credential's value to a plaintext file at rest.
Persistence SHALL use the operating system keyring, and only with the
operator's consent. When no keyring backend is available, RWR SHALL decline to
persist and say so, rather than fall back to a plaintext file - except the two
pre-existing built-in config-file paths for the GitHub token and SSH private
key, which SHALL be restricted to `0600` and SHALL warn with the file path when
used.

#### Scenario: Device-flow token persisted with a keyring available

- **WHEN** `--gh-auth` completes and an OS keyring backend is available
- **THEN** the token is saved to the keyring and no plaintext file gains the
  token value

#### Scenario: Generated SSH key persisted with a keyring available

- **WHEN** an SSH-key blueprint selects `set_as_rwr_ssh_key`
- **AND** an OS keyring backend is available
- **THEN** the base64 private key is saved under `ssh_private_key` in the keyring
- **AND** no plaintext file gains the key value
- **AND** RWR attempts to clear any older plaintext config value
- **AND** a cleanup failure is reported

#### Scenario: Built-in SSH key resolves from the keyring

- **GIVEN** no SSH key flag or config value is set
- **AND** the keyring contains `ssh_private_key`
- **WHEN** built-in credentials resolve
- **THEN** the keyring value becomes the SSH key used by git authentication

#### Scenario: SSH keyring backend is unavailable

- **WHEN** an SSH-key blueprint selects `set_as_rwr_ssh_key`
- **AND** no OS keyring backend is available
- **THEN** RWR warns that it is using the compatibility fallback
- **AND** writes the base64 key only to the owner-readable (`0600`) config file

#### Scenario: SSH keyring write fails while a backend exists

- **WHEN** an SSH-key blueprint selects `set_as_rwr_ssh_key`
- **AND** the keyring backend is locked, denies access, or returns a transient error
- **THEN** RWR reports the keyring error
- **AND** does not write `repository.ssh_private_key` to the config file
