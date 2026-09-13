# Credentials

Set up the machine first, then configure your vault and signing identity:

```sh
rwr run all --except credentials
rwr run scripts --except credentials
rwr run credentials --profile bitwarden
```

The root shorthands `rwr all` and `rwr credentials` work too. `--except` accepts repeated flags and comma-separated processor names. Unknown names and requesting/excluding the same named processor are errors. `blueprints.except` supplies full-run defaults; a named invocation overrides those defaults, while CLI exclusions always win.

## Acquisition policy

```yaml
blueprints:
  format: yaml
  location: .
  except: [credentials]
credentialPolicy:
  setup: explicit
  onUnavailable: skip
```

`setup: explicit` is the default: all runs do not onboard providers. To deliberately include setup in a full run, use `setup: ordered`, include credentials in the order, and remove its init exclusion. Consumers never trigger onboarding themselves.

**Behavior change:** declaring or exposing a credential no longer acquires it at startup. Runtime lookup is noninteractive. It never installs tools, logs in, unlocks, prompts, or saves values. Profiles are filtered before dependencies resolve. Missing values skip only dependent resources by default; `onUnavailable: fail` records their failure and continues independent work. Executed operation failures still make the run fail.

Excluding credentials denies managed provider calls, including probes. Already supplied environment values and noninteractive keyring reads remain available. Linux reads only unlocked Secret Service items; Windows uses Credential Manager. Runtime macOS keyring reads currently report unavailable to avoid an unexpected Keychain access dialog. Explicit keyring writes remain supported on all supported platforms.

## Trusted provider declarations

```yaml
credentialProviders:
  - name: personal-vault
    provider: bitwarden
    server: https://vault.bitwarden.com
    # account: me@example.com       # optional expected identity
    # sessionEnv: MY_BW_SESSION     # optional; default BW_SESSION
credentials:
  - name: gpg_passphrase
    scope: [credentials]
    sources: [env:RWR_CRED_GPG_PASSPHRASE, keyring]
    references:
      - connection: personal-vault
        item: gpg-signing
        field: password
credentialAttachments:
  - name: signing-private-key
    connection: personal-vault
    item: gpg-signing
    filename: private.asc
    write: true
```

Connections and item/attachment references belong in trusted init configuration. Blueprints select their names; they cannot redirect endpoints or authorize new attachments. `sources` retains its string syntax; the new optional `references` array is strictly decoded separately and is tried after local/legacy sources. Fields use the Bitwarden vocabulary: password, username, uri, notes, totp, or `field:NAME`. The first nonempty value wins.

Legacy `bw:ITEM/password`, `env:NAME`, `keyring`, and `prompt` declarations still parse. Legacy `bw:` reads the existing CLI context without onboarding. Prefer named connections for account isolation. `prompt` is never a runtime fallback; an explicitly selected native task may prompt when its declaration permits it. Native setup does not export a vault session into scripts or the parent shell.

## Setup blueprints

Put this under `credentials/`, in any supported format:

```yaml
credential_setup:
  - name: personal-signing
    profiles: [bitwarden, gpg-restore]
    connection: personal-vault
    install: if-missing
    session: ensure-ready
    tasks:
      - name: signing-key
        kind: gpg-restore
        source: signing-private-key
        fingerprint: YOUR_COMPLETE_KEY_FINGERPRINT
        passphrase: gpg_passphrase
        configureGitSigning: true
        # ownerTrust: 6             # explicit ultimate trust; omitted by default
```

`--profile bitwarden` is an ordinary profile filter, not a vendor selector. Imports work through `credential_setup: [{import: ../shared/credentials.yaml}]`. A setup entry without tasks installs/configures/authenticates only. `install` accepts `never` (default) or `if-missing`; `session` accepts `ensure-ready` (default) or `existing`. Missing tools or locked sessions in noninteractive mode report unavailable. Explicit authentication failures remain failures; choosing Skip suppresses later attempts during that run. Ctrl-C cancels.

Bitwarden uses an isolated CLI data directory under the OS config directory's `rwr/credential-providers/`, keyed by connection identity. Login supports the CLI's MFA interaction. An existing account/endpoint conflict is rejected. Installation uses the verified official CLI archive. Session tokens stay in the RWR process and child-only environments; closing a session does not lock/logout another application's vault. See the [official CLI documentation](https://bitwarden.com/help/cli/).

The shipped adapter is Bitwarden. Other vendors can implement the provider/session interfaces and optional attachment capability; 1Password and LastPass are not yet shipped adapters.

## Native tasks

- `gpg-restore`: checks the local identity before vault access, verifies the complete primary-key set and passphrase protection in a private temporary keyring, then imports into the invoking user's GPG home. Trust and Git commit signing are explicit options. GPG must already be installed.
- `gpg-backup`: exports protected private material, verifies it in a temporary keyring, uploads and downloads the new attachment for verification before deleting the old attachment. Requires a write-authorized binding and `writeProfile`, which must be explicitly selected. Optional `publicSource` and `revocationSource` bindings keep those artifacts separate; a missing local revocation certificate is omitted. A failed cleanup leaves both versions rather than deleting the verified backup.
- `keyring`: materializes the named `credential` into the OS keyring under a connection/account/credential namespace. Provider-sourced TOTP values cannot be persisted. This is an explicit write, not an automatic cache of every lookup.

Private keys, passphrases and sessions are never journaled. Status reports prior setup without claiming the vault is currently unlocked. Uninstall reports credentials as non-reversible; it does not delete keys, accounts or remote vault contents. Dry-run does not probe a provider, fetch/materialize secrets, or import/export/upload keys.

## Dependencies for scripts and files

```yaml
credentials:
  - name: deploy_token
    sources: [env:DEPLOY_TOKEN]
    scope: [scripts]
exposeCredentials: [deploy_token]
```

```yaml
scripts:
  - name: deploy
    action: run
    exec: self
    source: ./scripts
    profiles: [deploy]
    requiresCredentials: [deploy_token]
    onCredentialUnavailable: skip
```

The selected child gets `RWR_CRED_DEPLOY_TOKEN`; unrelated children do not. Declared source environment variables and provider session variables are withheld from ordinary child environments; use the scoped `RWR_CRED_` export in consumers. Scope/exposure are access permissions, not proof that a resource needs the secret. `requiresCredentials` and `onCredentialUnavailable` are supported on scripts, files, templates and directories. Simple `{{ .Credentials.name }}` references in inline content also declare demand; template files are scanned for credential references. Structural decoding preserves symbolic references until the consumer runs. Secrets cannot be substituted into metadata, paths or argv. Credential-bearing inline files default to mode 0600 and reject permissions for other users.

Opaque scripts that invoke `bw` themselves remain arbitrary user code. Migrate those to native credential tasks or declare their dependencies; RWR cannot infer vault calls from shell text. See [the native Bitwarden example](../examples/bitwarden/README.md).

### Where RWR stores a credential

RWR persists a managed credential only in the OS keyring - Secret Service on
Linux, Keychain on macOS, Credential Manager on Windows - and only when you
agree. By default, RWR does not write a managed credential to a plaintext file.

The two built-in credentials retain their grandfathered config-file fallback:
`repository.gh_api_token` and `repository.ssh_private_key` keep working. New
GitHub tokens and keys selected by `set_as_rwr_ssh_key` go to the keyring first;
when no keyring backend is available, RWR falls back to the config file at
`0600` and warns with the file path. When a generated SSH key moves into the
keyring successfully, RWR attempts to clear an older plaintext config value so
it cannot override the new key on the next run. A cleanup failure is reported.

## How to permit a credential

Some blueprints need a credential. Examples are a blueprint that writes a
`.netrc` file, a blueprint that configures `gh`, and a script that calls the
GitHub API.

Give the name of each credential that the blueprints can read:

```yaml
blueprints:
  format: yaml
  location: "."

exposeCredentials:
  - gh_api_token
```

RWR shares only the credentials that you give. The name `gh_api_token` does not
give access to the SSH key.

These names are correct:

| Name | RWR also accepts |
|---|---|
| `gh_api_token` | `repository.gh_api_token` |
| `ssh_private_key` | `repository.ssh_private_key` |
| `bw_session` | Legacy built-in name; native provider sessions are private and are never published through this credential |

RWR gives a warning at start when a credential is available. The change is
always visible.

## What a permitted credential gives you

A declared credential appears in a template as `{{ .Credentials.<name> }}` and
in a script as `RWR_CRED_<NAME>`:

```text
password={{ .Credentials.deploy_token }}
```

Template content can receive the exposed value. Scripts should use the declared child environment and pass values to tools through stdin or a supported secret environment variable. Do not place secret values in command arguments.

The two built-in credentials keep their original names. In a template:

```yaml
templates:
  - name: netrc
    action: copy
    source: ./src/netrc.tmpl
    target: "{{ .User.home }}/.netrc"
```

```
machine github.com login {{ .User.username }} password {{ .Flags.ghAPIToken }}
```

In a script, through the environment:

```bash
#!/usr/bin/env bash
gh auth login --with-token <<< "$RWR_VAR_REPOSITORY_GH_API_TOKEN"
```

## How to keep the risk small

- Give only the credential that the blueprint needs.
- Set `exposeCredentials` in the init file of the tree that needs it. Do not set
  it in an init file that other trees use.
- Let RWR do the work when it can. An `ssh_keys` blueprint sends the key to
  GitHub with the token. The blueprint does not read the token.

## Logs

RWR removes credential values from the logs. This applies to permitted
credentials also.

If you must see a value in the logs, use the `--show-secrets` flag. RWR gives a
warning while this flag is active.
