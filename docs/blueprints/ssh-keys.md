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

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes | The name of the SSH key file (e.g., `id_rsa`) |
| `type` | No | The type of the SSH key (e.g., `rsa`, `dsa`, `ecdsa`, `ed25519`). Default is `rsa` |
| `path` | No | The directory where the SSH key will be stored. Default is `~/.ssh` |
| `comment` | No | A comment to include in the SSH key (e.g., email address) |
| `no_passphrase` | No | Set to `true` to generate the SSH key without a passphrase. Default is `false` |
| `copy_to_github` | No | Set to `true` to copy the public key to your GitHub account. Default is `false` |
| `github_title` | No | The title to use for the SSH key when copying it to GitHub |
| `set_as_rwr_ssh_key` | No | Set to `true` to use this key as the default RWR SSH key. Default is `false` |

## Generating SSH Keys

When the SSH Keys blueprint is processed, RWR will generate the specified SSH keys using the provided settings. The keys will be stored in the specified `path` directory.

If `no_passphrase` is set to `true`, the SSH key will be generated without a passphrase. Otherwise, RWR will prompt you to enter a passphrase for the key.

## Copying Public Keys to GitHub

If `copy_to_github` is set to `true`, RWR will attempt to copy the public key to your GitHub account. To use this feature, you need to provide a GitHub API token with the necessary permissions.

You can set the GitHub API token using the `--gh-api-key` flag when running RWR or by configuring it in the `config.yaml` file under the `repository.gh_api_token` setting.

If `github_title` is provided, it will be used as the title for the SSH key on GitHub. If not specified, the hostname of the machine will be used as the title.

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
