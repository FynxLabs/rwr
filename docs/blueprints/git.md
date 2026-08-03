# Git Blueprint

The Git blueprint in Rinse, Wash, Repeat (RWR) allows you to clone and manage Git repositories as part of your system configuration. This page describes how to define and use the Git blueprint.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

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
| `name` | Yes, if `import` is not provided | A unique name for the repository |
| `import` | Yes, if `name` is not provided | Path to import git repository definitions from another file (relative to blueprint directory) |
| `action` | No | `clone` or `pull`. Empty or `clone` clones when `path` does not exist and pulls when it does; `pull` requires something to already be checked out at `path`. Any other value is an error |
| `url` | Yes | The URL of the Git repository to clone |
| `branch` | No | The branch to clone (defaults to the repository's default branch) |
| `path` | Yes | The local path where the repository should be cloned |
| `private` | No | Indicates whether the repository is private (defaults to `false`) |
| `profiles` | No | List of profiles this repository belongs to. If empty, repository is always cloned (base item) |
| `interactive` | No | Accepted by the schema but **not read** by the git processor: clones and pulls run the same way whatever it is set to |

## Blueprint Imports

Import git repository definitions from other files:

```yaml
git:
  # Import shared repositories
  - import: ../../Common/git/base-repos.yaml

  # Add project-specific repositories
  - name: my-project
    action: clone
    url: https://github.com/username/my-project.git
    path: ~/projects/my-project
    profiles:
      - dev
```

This allows you to maintain common repository lists separately from project-specific ones.

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

If the issue persists, open an issue at [github.com/fynxlabs/rwr/issues](https://github.com/fynxlabs/rwr/issues).
