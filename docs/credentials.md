# Credentials in blueprints

RWR manages credentials for your blueprints. GitHub API tokens, SSH private keys, and RWR-created Bitwarden sessions are
built in; the init file can declare more. By
default, blueprints cannot read any of them.

## Why RWR holds back the credentials

Blueprints usually come from a git repository. Everything that a blueprint can
read, the author of that blueprint can read:

- A template reads from a blueprint file. RWR writes the result to a path that
  the same blueprint selects. A token in template scope can go to any path.
- A script gets the environment of RWR. Every script in the blueprint can read
  each variable in that environment.

This is safe when the blueprint is yours. This is not safe for a blueprint from a
different author. RWR cannot tell the difference between the two.

## Declare the credentials that a tree needs

A blueprint sometimes needs a secret that is not the GitHub token - a registry
token, an API key, a license key. Declare it in the init file, and RWR finds
the value for you:

```yaml
credentials:
  - name: cachix_token
    description: "Cachix auth token for the nix cache"
    sources: [env:CACHIX_AUTH_TOKEN, keyring, prompt]
    scope: [scripts]
```

| Field | Description | Required |
|---|---|---|
| `name` | The identity of the credential, everywhere: in `exposeCredentials`, in `{{ .Credentials.<name> }}`, in `RWR_CRED_<NAME>`, and in the keyring. A lowercase identifier. | Yes |
| `sources` | The ordered places to look: `env:<VAR>`, `bw:<item>[/<key>]`, `keyring`, `prompt`. The first source with a value wins. | No. The default is `[env:RWR_CRED_<NAME>, keyring, prompt]` |
| `description` | Text that the prompt shows | No |
| `scope` | Where the credential goes after you expose it: `scripts` (the command environment), `templates` (template rendering), or a processor name (the credential resolves only when that processor runs) | No. The default is everywhere that `exposeCredentials` reaches |

### Reading a credential from Bitwarden

A `bw:` source reads from your personal vault through the Bitwarden CLI
(`bw`). If it is missing, an interactive run offers **Install** or **Skip**.
RWR downloads the official standalone binary, verifies its SHA-256 digest, and
installs it in its user configuration directory under `rwr/bin`. No npm,
Homebrew, curl, unzip, or administrator access is needed. The managed binary
is available to RWR and its scripts on subsequent runs.

Skipping, running non-interactively, or an installation failure leaves vault
credentials unset and continues the run. Environment and keyring fallbacks
are still tried; RWR does not force a password prompt for a missing CLI.
RWR handles login and unlock in the same run, including server selection and
Bitwarden MFA prompts. You do not need to run shell commands or export a session.
The master password is passed only to Bitwarden and is never saved. The syntax is
`bw:<item>[/<key>]`, where `<item>` is anything `bw get` accepts (an item ID
or a name) and `<key>` is one of:

| Key | Reads |
|---|---|
| `password` (or nothing) | the item's password field |
| `username` | the username |
| `uri` | the first URI |
| `notes` | the notes field |
| `totp` | the current TOTP code, not the seed |
| `field:<name>` | a custom field by exact name - text or hidden |

For attachment scripts that need to call `bw` themselves, add `bw_session` to
`exposeCredentials` and use `export BW_SESSION="$RWR_CRED_BW_SESSION"` inside
the script. This explicitly grants those scripts access to the unlocked vault.
RWR does not put the session in the environment unless the blueprint opts in.
Existing `BW_SESSION` values still work for unattended runs.

An item name may itself contain slashes: only a final segment that names a
key is read as the key, so `bw:org/team` reads the password of item
`org/team` and `bw:org/team/username` reads its username. A name that would
collide - one ending in `/password`, for instance - cannot be addressed by
name; use the item ID.

A caveat on `totp`: `bw get totp` returns the current verification code,
which expires about 30 seconds after `bw` generates it - and rwr resolves
every credential before any processor runs, so a script that runs later in
the run may receive an already-expired code. It suits a script that consumes
the code immediately, not anything that needs a long-lived seed (the seed is
not retrievable through `bw get` at all). Reading TOTP also requires a paid
Bitwarden plan (Premium or an organization).

```yaml
credentials:
  - name: cachix_token
    description: "Cachix auth token for the nix cache"
    sources: [bw:cachix/field:api-key, keyring, prompt]
    scope: [scripts]
```

The value arrives through the same gates as any other credential: it stays
out of templates and script environments until `exposeCredentials` names it,
and the logs redact it.

A `bw:` source that cannot yield a value - CLI missing, vault locked, no such
item - is a miss, not an error: resolution moves on to the next source, and
the reason is in the log. That is what makes an order like
`bw:..., keyring, prompt` useful: a machine with the vault locked still
resolves the credential from the keyring or a prompt. One CLI call has a 30
second timeout, and stdin is never attached, so a first-run wizard cannot
hang the run. Attachments are not readable through a `bw:` source; a scripts
blueprint is the tool for files - see
[examples/bitwarden](../examples/bitwarden/README.md) for a complete GPG key
backup/restore tree.

RWR resolves every declared credential at the start of the run, before any
processor runs. Except when Bitwarden was missing and skipped as described
above, a credential with no value from any source stops the run with an error that names the credential and the sources tried. When the run is not
interactive - no terminal, or `--interactive=false` - RWR skips `prompt` and
the error says so.

After you answer a prompt, RWR offers to save the value to the OS keyring, so
the next run does not ask again.

A declared credential gets the same protection as the built-in two: RWR keeps
it out of template scope and out of the command environment until you name it
under `exposeCredentials`, and the logs show `[redacted]` instead of the value.

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

RWR gives a warning at start when a credential is available. The change is
always visible.

## What a permitted credential gives you

A declared credential appears in a template as `{{ .Credentials.<name> }}` and
in a script as `RWR_CRED_<NAME>`:

```
cachix authtoken {{ .Credentials.cachix_token }}
```

```bash
#!/usr/bin/env bash
cachix authtoken "$RWR_CRED_CACHIX_TOKEN"
```

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
