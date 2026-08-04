# Change: Managed auth - declared credentials, sourced and redacted

## Why

rwr's credential story is two hardcoded secrets and ad-hoc plumbing around one
of them. `types.SecretConfigKeys` (`internal/types/secrets.go:15-18`) names
exactly `repository.gh_api_token` and `repository.ssh_private_key`; everything
downstream - withholding from template scope, the `RWR_VAR_*` export gate
(`internal/processors/initialize.go:239-244`), log redaction - keys off that
fixed list. A blueprint that needs any *other* secret (a registry token, an API
key a script writes into a `.netrc`, a license key) has no declared path at
all: the operator smuggles it in as an `RWR_*` environment variable, which
lands in `Variables.UserDefined` (`initialize.go:227-233`) with none of the
secret handling - unredacted in debug logs, exported to every spawned command.

The GitHub token shows what the pieces look like when hand-built: `--gh-auth`
runs an OAuth device flow (`internal/processors/ssh_key.go:278`,
`AuthenticateWithGitHub`), `--gh-api-key` takes it on the command line
(`cmd/root.go:145`), and the result persists via `viper.WriteConfig` into the
config file - plaintext at rest, guarded only by 0600 permissions
(`internal/prompts/github.go`, `SaveGitHubTokenToConfig`). None of that
machinery is reusable for a second credential; the ssh_keys processor is the
only one that gets device-flow pre-work (`cmd/run.go:110-123`).

`exposeCredentials` (`internal/types/initialize.go:67-70`) already gives trees
an opt-in vocabulary for *sharing* credentials with blueprints. What is missing
is the other half: a way for a tree to *declare* the credentials it needs, and
for rwr to *source* them - so the declared path is also the safe path.

## What Changes

- A `credentials:` section in the init file. Each entry declares a named
  credential the tree's blueprints need:
  - `name` - the identifier blueprints reference (e.g. `cachix_token`)
  - `sources` - ordered list of places to look: `env:<VAR>`, `keyring`,
    `prompt` (interactive input, masked), with a documented default order
  - `description` - shown when prompting
  - optional `scope` - which processors may read it (default: templates and
    scripts, same surface `exposeCredentials` governs today)
- Resolution at init time, after config load and before any processor runs:
  first source that yields a value wins; a declared credential that resolves
  nowhere is an error naming the credential and the sources tried (non-TTY
  runs cannot prompt, so `prompt` is skipped there and the error says so).
- Every managed credential gets the full secret treatment that today only the
  two hardcoded keys get: withheld from template scope and `RWR_VAR_*` export
  unless named in `exposeCredentials`, redacted in logs behind
  `types.RedactedPlaceholder`, never placed in argv (stdin or file at 0600
  only, matching the existing convention in `openspec/config.yaml`).
- Storage: opt-in `keyring` persistence via the OS keychain (Secret Service /
  macOS Keychain / Windows Credential Manager). rwr SHALL never write a
  managed credential to a plaintext file at rest. The existing GitHub-token
  config-file write is grandfathered but the device flow learns to prefer the
  keyring when available.
- The GitHub token becomes the first managed credential: `--gh-auth` /
  `--gh-api-key` become sugar for sourcing `gh_api_token`, keeping their flags
  and behavior. `exposeCredentials` keeps its exact semantics, now applying to
  the whole declared set instead of a two-entry hardcoded list.

## Breakage

Nothing breaks for existing blueprints and init files. `credentials:` is a new
optional section; trees without it behave exactly as today, including the two
built-in secrets, `exposeCredentials`, `--gh-auth`, and `--gh-api-key`. The
device flow preferring the keyring changes where a *new* token is persisted;
an existing config-file token keeps being read.

## Impact

- Affected specs: `credential-handling` (the bulk), `initialization` (new init
  section), `cli` (flag semantics folded into sourcing).
- Affected code: `internal/types/secrets.go` (dynamic secret set),
  `internal/processors/initialize.go` (resolution phase),
  `internal/prompts/github.go` / `internal/processors/ssh_key.go` (device flow
  re-pointed at the credential store), new keyring package.
- Security posture: blueprints remain untrusted; a blueprint can *reference* a
  declared credential but never *declare* one - declarations live only in the
  init file the operator points rwr at, and exposure still requires the
  operator's `exposeCredentials` opt-in.
