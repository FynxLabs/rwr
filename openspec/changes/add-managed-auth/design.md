# Design: Managed auth

## Context

Today's secret handling is a fixed two-key list (`internal/types/secrets.go`)
with three behaviors attached: template-scope withholding, `RWR_VAR_*` export
gating, and log redaction. The design generalizes the *list* while keeping the
three behaviors exactly as specified in `credential-handling` — they are
already right, they just apply to too few things.

## Goals

- One declared path for every secret a tree needs; the declared path is the
  safe path.
- No plaintext secrets at rest, ever, for managed credentials.
- The trust model is unchanged: init file declares, operator exposes,
  blueprints only consume.

## Non-goals

- Secret *management* (rotation, expiry, vault backends). Sources are local:
  env, OS keyring, prompt. A future `exec:` source could shell out to
  `pass`/`op`/`vault` — out of scope here, but the sources list is ordered and
  extensible precisely so that can land without schema change.
- Per-blueprint scoping. Scope is per-processor at most; anything finer needs
  provenance tracking rwr does not have.

## Decisions

### Declaration shape

```yaml
credentials:
  - name: cachix_token
    description: "Cachix auth token for the nix cache"
    sources: [env:CACHIX_AUTH_TOKEN, keyring, prompt]
    scope: [scripts]        # optional; default: wherever exposeCredentials reaches
```

- `name` is the identity everywhere: in `exposeCredentials`, in template scope
  (`{{ .Credentials.cachix_token }}`), in the exported env name
  (`RWR_CRED_CACHIX_TOKEN`), in the keyring entry.
- Built-ins `gh_api_token` and `ssh_private_key` are implicit declarations with
  sources `[flag, config, env, keyring, prompt]` so existing behavior is a
  special case, not a parallel system. `types.SecretConfigKeys` stays as the
  bridge from viper keys to credential names.

### Source precedence

Ordered, first hit wins, per credential (the declaration's order is the
precedence). Default when `sources` is omitted: `[env:RWR_CRED_<NAME>, keyring,
prompt]`. Rationale: env first so CI can inject without prompting; keyring
before prompt so an interactive answer given once (and saved) is not re-asked.

`prompt`:
- masked input (same huh machinery as `PromptGitHubToken`,
  `internal/prompts/github.go`)
- skipped when not a TTY or `--interactive=false`; the resolution error then
  names the sources that were actually tried
- after a successful prompt, offer (interactive, default yes) to save to the
  keyring so next run doesn't ask

### Resolution timing

A single resolution phase in `ProcessInitialization` after `SetExposedCredentials`
(`internal/processors/initialize.go:156`) and before
`setUserDefinedAndEnvVariables`, so the export gate sees the full credential
set. All declared credentials resolve up front — failing at minute 20 inside a
script is worse than failing at second 1 — except credentials scoped to
processors not selected for this run, which are skipped.

### Storage

- Keyring only, via the OS-native store (service `rwr`, account = credential
  name). Candidate library: `zalando/go-keyring` (no cgo, covers Secret
  Service, macOS Keychain, Windows Credential Manager). Headless Linux without
  a Secret Service provider: `keyring` source resolves to "unavailable" and
  precedence moves on — never an error on read, loud error on an explicit save.
- Hard rule: rwr never writes a managed credential to a plaintext file. The
  one existing violation — the GitHub token in the viper config file
  (`SaveGitHubTokenToConfig`) — is grandfathered for reads; the device flow's
  save path prefers the keyring when available and only falls back to the
  config file with a warning naming the file.

### Redaction

Managed credential values register with the existing redaction machinery at
resolution time, so every value is behind `types.RedactedPlaceholder` in logs
and `--show-secrets` (`cmd/root.go:178`) reveals them uniformly. Values must
never appear in argv (readable via `ps`, per `openspec/config.yaml`): the
export surface is env (opt-in via `exposeCredentials`) and template scope
(same opt-in) only.

### Scope

`scope` limits which processors see the credential even after exposure:
`scripts`, `templates` (files/templates rendering), or a processor name.
Enforcement point is the same gate that today checks `IsCredentialExposed` —
the check becomes (exposed AND in scope for the consuming processor). Default
scope is everything `exposeCredentials` reaches today, so omitting `scope`
reproduces current semantics.

## Risks

- Keyring behavior varies wildly across desktop environments; mitigation:
  keyring is never load-bearing (always another source in the chain) and the
  test suite fakes the keyring interface.
- Two names for one secret (viper key vs credential name) during transition;
  mitigation: `normaliseCredentialKey` already collapses these
  (`internal/types/secrets.go:70-77`), keep it the single normalizer.
