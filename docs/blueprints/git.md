# Git Blueprint

The Git blueprint in Rinse, Wash, Repeat (RWR) allows you to clone and manage Git repositories as part of your system configuration. This page describes how to define and use the Git blueprint.

## Blueprint Structure

The Git blueprint follows a specific structure to define the repositories to be cloned and managed. Here's an example of a Git blueprint in YAML format:

```yaml
git:
  - name: my-repo
    action: clone
    url: https://github.com/username/my-repo.git
    branch: main
    path: /path/to/clone/my-repo
    private: false
  - name: private-repo
    action: clone
    url: git@github.com:username/private-repo.git
    branch: develop
    path: /path/to/clone/private-repo
    private: true
```

## Blueprint Settings

The following settings are available for each repository in the Git blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes | A unique name for the repository |
| `action` | Yes | The action to perform (`clone` is the only supported action) |
| `url` | Yes | The URL of the Git repository to clone |
| `branch` | No | The branch to clone (defaults to the repository's default branch) |
| `path` | Yes | The local path where the repository should be cloned |
| `private` | No | Indicates whether the repository is private (defaults to `false`) |

## Private Repositories

To clone private repositories, you need to provide authentication details. RWR supports two authentication methods:

1. GitHub API Key: Set the `--gh-api-key` flag or configure the `repository.gh_api_token` setting in the configuration file.
2. SSH Key: Set the `--ssh-key` flag or configure the `repository.ssh_private_key` setting in the configuration file. The SSH key should be base64 encoded.

## Examples

Here are a few examples of using the Git blueprint in different formats:

### YAML

```yaml
git:
  - name: my-repo
    action: clone
    url: https://github.com/username/my-repo.git
    path: /path/to/clone/my-repo
```

### JSON

```json
{
  "git": [
    {
      "name": "my-repo",
      "action": "clone",
      "url": "https://github.com/username/my-repo.git",
      "path": "/path/to/clone/my-repo"
    }
  ]
}
```

### TOML

```toml
[[git]]
name = "my-repo"
action = "clone"
url = "https://github.com/username/my-repo.git"
path = "/path/to/clone/my-repo"
```

## Troubleshooting

If you encounter issues while using the Git blueprint, consider the following:

- Ensure that the repository URL is correct and accessible.
- Verify that you have provided the necessary authentication details for private repositories.
- Check that the specified local path for cloning the repository is valid and has the required permissions.

If the issue persists, please refer to the [Troubleshooting](../troubleshooting.md) section or reach out to the RWR community for assistance.
