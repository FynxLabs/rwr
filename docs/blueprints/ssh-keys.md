# SSH Keys Blueprint

The SSH Keys blueprint in Rinse, Wash, Repeat (RWR) allows you to generate and manage SSH keys as part of your system configuration. You can create SSH keys, specify their properties, optionally copy the public keys to your GitHub account, and set a key as the default RWR SSH key.

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
| `name`               | Yes      | The name of the SSH key file (e.g., `id_rsa`)                                      |
| `type`               | No       | The type of the SSH key (e.g., `rsa`, `dsa`, `ecdsa`, `ed25519`). Default is `rsa` |
| `path`               | No       | The directory where the SSH key will be stored. Default is `~/.ssh`                |
| `comment`            | No       | A comment to include in the SSH key (e.g., email address)                          |
| `no_passphrase`      | No       | Set to `true` to generate the SSH key without a passphrase. Default is `false`     |
| `copy_to_github`     | No       | Set to `true` to copy the public key to your GitHub account. Default is `false`    |
| `github_title`       | No       | The title to use for the SSH key when copying it to GitHub                         |
| `set_as_rwr_ssh_key` | No       | Set to `true` to use this key as the default RWR SSH key. Default is `false`       |

## Generating SSH Keys

When the SSH Keys blueprint is processed, RWR will generate the specified SSH keys using the provided settings. The keys will be stored in the specified `path` directory.

If `no_passphrase` is set to `true`, the SSH key will be generated without a passphrase. Otherwise, RWR will prompt you to enter a passphrase for the key.

## Copying Public Keys to GitHub

If `copy_to_github` is set to `true`, RWR will attempt to copy the public key to your GitHub account.

### GitHub Authentication

RWR supports three methods for GitHub authentication (in priority order):

1. **`--gh-api-key` / `--gh-key` flag** - Provide an explicit GitHub token
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
4. Save the token to your RWR config

After this initial setup, future runs won't require `--gh-auth` as the token is saved in your config.

#### Using an Explicit Token

```bash
rwr run ssh_keys --gh-key ghp_your_token_here
```

Or use the longer form:

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
- Or use `--gh-key` flag with your token
- Or set `GITHUB_TOKEN` environment variable

#### Authentication timeout

- You have 5 minutes to authorize after running `--gh-auth`
- Run the command again to get a new code

#### Authentication failed: invalid GitHub API token

- Token may have expired
- Re-authenticate with `--gh-auth`
- Or generate a new token with `write:public_key` scope

## Setting the RWR SSH Key

If `set_as_rwr_ssh_key` is set to `true`, RWR will set this key as the default SSH key for RWR operations. This key will be used for private git clones and other SSH-based operations within RWR. The private key will be base64 encoded and stored in the RWR configuration file.

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
