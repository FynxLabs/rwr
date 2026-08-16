# SSH Keys Blueprint

The SSH Keys blueprint in Rinse, Wash, Repeat (RWR) allows you to generate and manage SSH keys as part of your system configuration. You can create SSH keys, specify their properties, optionally copy the public keys to your GitHub account, and set a key as the default RWR SSH key.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

The SSH Keys blueprint has the following structure:

```yaml
ssh_keys:
  - name: id_rsa
    type: rsa
    path: ~/.ssh
    comment: john@example.com
    no_passphrase: true
    copy_to_github: true
    github_title: My SSH Key
    set_as_rwr_ssh_key: false
```

## Blueprint Settings

The following settings are available for each SSH key in the SSH Keys blueprint:

| Setting              | Required | Description                                                                        |
| -------------------- | -------- | ---------------------------------------------------------------------------------- |
| `name`               | Yes, if `import` is not provided      | The name of the SSH key file (e.g., `id_rsa`)                                      |
| `import`             | Yes, if `name` is not provided | Path to import SSH key definitions from another file (relative to blueprint directory) |
| `type`               | Yes      | The key type, passed straight to `ssh-keygen -t`. There is **no default** - omitting it makes `ssh-keygen` fail. `rwr validate` warns for anything other than `rsa`, `ed25519` or `ecdsa` |
| `path`               | Yes      | The directory where the SSH key will be stored. There is **no default**; write it explicitly, e.g. `~/.ssh`. A leading `~` is expanded |
| `comment`            | No       | A comment to include in the SSH key (e.g., email address)                          |
| `no_passphrase`      | No       | Set to `true` to generate the SSH key without a passphrase. Default is `false`     |
| `copy_to_github`     | No       | Set to `true` to copy the public key to your GitHub account. Default is `false`    |
| `github_title`       | No       | The title to use for the SSH key when copying it to GitHub                         |
| `set_as_rwr_ssh_key` | No       | Set to `true` to use this key as the default RWR SSH key. Default is `false`       |
| `profiles`           | No       | List of profiles this SSH key belongs to. If empty, key is always generated (base item) |
| `interactive`        | No       | Override global interactive mode for this SSH key (`true`/`false`). If omitted, uses the global `--interactive` flag. Useful for ensuring passphrase prompts appear even in non-interactive mode |

## Blueprint Imports

Import SSH key definitions from other files:

```yaml
ssh_keys:
  # Import common SSH keys
  - import: ../../Common/ssh_keys/base-keys.yaml

  # Add machine-specific keys
  - name: id_ed25519_work
    type: ed25519
    path: ~/.ssh
    comment: work@example.com
    copy_to_github: true
    github_title: Work Machine Key
    profiles:
      - work
```

This allows you to share SSH key configurations across multiple machines.

## Generating SSH Keys

When the SSH Keys blueprint is processed, RWR will generate the specified SSH keys using the provided settings. The keys will be stored in the specified `path` directory.

If `no_passphrase` is set to `true`, the SSH key will be generated without a passphrase. Otherwise, RWR will prompt you to enter a passphrase for the key.

A key file that already exists is left alone: RWR logs a warning, skips
generation, and goes on to the `copy_to_github` and `set_as_rwr_ssh_key` steps
with the existing key.

## Copying Public Keys to GitHub

If `copy_to_github` is set to `true`, RWR will attempt to copy the public key to your GitHub account.

### GitHub Authentication

RWR supports three methods for GitHub authentication (in priority order):

1. **`--gh-api-key` flag** - Provide an explicit GitHub token (`--gh-key` is a deprecated alias)
2. **`--gh-auth` flag** - Authenticate using OAuth device flow (recommended for first-time setup)
3. **`GITHUB_TOKEN` environment variable** - For CI/CD environments

#### First Time Setup - OAuth Authentication

```bash
rwr run ssh_keys --gh-auth
```

This will:

1. Display a device code (e.g., `ABCD-1234`)
2. Prompt you to visit <https://github.com/login/device>
3. Wait for you to authorize the application
4. Save the token to the OS keyring, or to the owner-only (`0600`) RWR config
   only when no keyring backend is available

After this initial setup, future runs won't require `--gh-auth` because the
token is persisted in one of those stores.

#### Using an Explicit Token

```bash
rwr run ssh_keys --gh-api-key ghp_your_token_here
```

#### Using Environment Variable (CI/CD)

```bash
export GITHUB_TOKEN=ghp_your_token_here
rwr run ssh_keys
```

### Token Requirements

The GitHub token needs the `write:public_key` scope to upload SSH keys.

### GitHub Key Title

If `github_title` is provided, it will be used as the title for the SSH key on GitHub. If not specified, the hostname of the machine will be used as the title.

### Troubleshooting

#### GitHub token not found

- Use `--gh-auth` to authenticate via OAuth
- Or use the `--gh-api-key` flag with your token
- Or set `GITHUB_TOKEN` environment variable

#### Authentication timeout

- You have 5 minutes to authorize after running `--gh-auth`
- Run the command again to get a new code

#### Authentication failed: invalid GitHub API token

- Token may have expired
- Re-authenticate with `--gh-auth`
- Or generate a new token with `write:public_key` scope

## Setting the RWR SSH Key

If `set_as_rwr_ssh_key` is set to `true`, RWR will set this key as the default SSH key for RWR operations. This key will be used for private git clones and other SSH-based operations within RWR. The private key is base64 encoded and saved to the OS keyring. If no keyring backend is available, RWR warns and falls back to the owner-only (`0600`) configuration file for compatibility with existing installations.

> [!NOTE]
> Only one key should be set as the RWR SSH key. If multiple keys are set, the last one processed will be used.

## Example

Here's an example of using the SSH Keys blueprint in YAML format:

```yaml
ssh_keys:
  - name: id_rsa
    type: rsa
    path: ~/.ssh
    comment: john@example.com
    no_passphrase: true
    copy_to_github: true
    github_title: My SSH Key
    set_as_rwr_ssh_key: true

  - name: id_ed25519
    type: ed25519
    path: ~/.ssh
    comment: john@example.com
    no_passphrase: false
    copy_to_github: false
```

In this example, two SSH keys are defined: `id_rsa` and `id_ed25519`. The `id_rsa` key is generated without a passphrase, copied to GitHub with the title "My SSH Key", and set as the default RWR SSH key. The `id_ed25519` key is generated with a passphrase and not copied to GitHub or set as the RWR SSH key.

For more information on using the SSH Keys blueprint in your RWR configuration, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) sections of the documentation.
