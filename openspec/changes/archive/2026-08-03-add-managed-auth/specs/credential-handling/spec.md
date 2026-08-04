# Credential Handling - Deltas

## ADDED Requirements

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
persist and say so, rather than fall back to a plaintext file - except the
pre-existing GitHub-token config-file path, which SHALL warn with the file
path when used.

#### Scenario: Device-flow token persisted with a keyring available

- **WHEN** `--gh-auth` completes and an OS keyring backend is available
- **THEN** the token is saved to the keyring and no plaintext file gains the
  token value
