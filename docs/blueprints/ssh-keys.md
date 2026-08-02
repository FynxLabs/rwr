# SSH Keys Blueprint

With the SSH Keys blueprint in Rinse, Wash, Repeat (RWR), you generate and manage SSH keys as part of your system configuration:

- create SSH keys and set their properties
- copy the public keys to your GitHub account
- set one key as the default RWR SSH key

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
| `name`               | Yes, if `import` is not provided      | The name of the SSH key file (for example, `id_rsa`)                                      |
| `import`             | Yes, if `name` is not provided | Path to import SSH key definitions from another file (relative to blueprint directory) |
| `type`               | Yes      | The key type, passed straight to `ssh-keygen -t`. There is **no default** — omitting it makes `ssh-keygen` fail. `rwr validate` warns for anything other than `rsa`, `ed25519` or `ecdsa` |
| `path`               | Yes      | The directory where RWR stores the SSH key. There is **no default**. Write it explicitly, for example `~/.ssh`. RWR expands a leading `~` |
| `comment`            | No       | A comment to include in the SSH key (for example, an email address)                          |
| `no_passphrase`      | No       | Set to `true` to generate the SSH key without a passphrase. Default is `false`     |
| `copy_to_github`     | No       | Set to `true` to copy the public key to your GitHub account. Default is `false`    |
| `github_title`       | No       | The title to use for the SSH key when copying it to GitHub                         |
| `set_as_rwr_ssh_key` | No       | Set to `true` to use this key as the default RWR SSH key. Default is `false`       |
| `profiles`           | No       | List of profiles this SSH key belongs to. If empty, key is always generated (base item) |
| `interactive`        | No       | Override global interactive mode for this SSH key (`true`/`false`). If omitted, uses the global `--interactive` flag. Use it to make sure that passphrase prompts appear in non-interactive mode |

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

You can then share SSH key configurations across multiple machines.

## Generating SSH Keys

RWR generates the SSH keys with the settings you give. It stores the keys in the `path` directory.

If `no_passphrase` is `true`, RWR generates the SSH key without a passphrase. Otherwise, RWR prompts you for a passphrase.

RWR does not change a key file that already exists:

- RWR logs a warning and skips generation.
- It goes on to the `copy_to_github` and `set_as_rwr_ssh_key` steps with the
  existing key.

## Copying Public Keys to GitHub

If `copy_to_github` is `true`, RWR copies the public key to your GitHub account.

### GitHub Authentication

RWR supports three methods for GitHub authentication (in priority order):

1. **`--gh-api-key` / `--gh-key` flag** - Provide an explicit GitHub token
2. **`--gh-auth` flag** - Authenticate using OAuth device flow (recommended for first-time setup)
3. **`GITHUB_TOKEN` environment variable** - For CI/CD environments

#### First Time Setup - OAuth Authentication

```bash
rwr run ssh_keys --gh-auth
```

This command:

1. Displays a device code (for example, `ABCD-1234`)
2. Prompts you to visit <https://github.com/login/device>
3. Waits for you to authorize the application
4. Saves the token to your RWR config

After this initial setup, future runs do not require `--gh-auth`, because the token is saved in your config.

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

If you give `github_title`, RWR uses it as the title for the SSH key on GitHub. If not, RWR uses the hostname of the machine as the title.

### Troubleshooting

#### GitHub token not found

- Use `--gh-auth` to authenticate via OAuth
- Or use `--gh-key` flag with your token
- Or set `GITHUB_TOKEN` environment variable

#### Authentication timeout

- You have 5 minutes to authorize after running `--gh-auth`
- Run the command again to get a new code

#### Authentication failed: invalid GitHub API token

- A possible cause is an expired token
- Re-authenticate with `--gh-auth`
- Or generate a new token with `write:public_key` scope

## Setting the RWR SSH Key

If `set_as_rwr_ssh_key` is `true`, RWR sets this key as the default SSH key for RWR operations. RWR uses this key for private git clones and other SSH operations. RWR encodes the private key in base64 and stores it in the RWR configuration file.

> [!NOTE]
> Set only one key as the RWR SSH key. If multiple keys are set, RWR uses the last one processed.

## Example

Here is an example of the SSH Keys blueprint in YAML format:

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

This example defines two SSH keys:

- `id_rsa`: generated without a passphrase, copied to GitHub with the title
  "My SSH Key", and set as the default RWR SSH key.
- `id_ed25519`: generated with a passphrase. Not copied to GitHub and not set
  as the RWR SSH key.

For more information, see the [Blueprints Overview](../blueprints-general.md) and [Best Practices](../best-practices.md).
