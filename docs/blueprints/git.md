# Git Blueprint

With the Git blueprint in Rinse, Wash, Repeat (RWR), you clone and manage Git repositories as part of your system configuration. This page describes how to define and use the Git blueprint.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

The Git blueprint defines the repositories to clone and manage. Here is an example of a Git blueprint in YAML format:

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
| `action` | No | Accepted by the schema but **not read**. RWR clones each entry if `path` does not exist and pulls it if it does |
| `url` | Yes | The URL of the Git repository to clone |
| `branch` | No | The branch to clone (defaults to the repository's default branch) |
| `path` | Yes | The local path where RWR clones the repository |
| `private` | No | Indicates whether the repository is private (defaults to `false`) |
| `profiles` | No | List of profiles this repository belongs to. If empty, repository is always cloned (base item) |
| `interactive` | No | Override global interactive mode for this git repository (`true`/`false`). If omitted, uses the global `--interactive` flag |

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

You can then keep common repository lists separate from project-specific ones.

## Private Repositories

To clone private repositories, you need to provide authentication details. RWR supports two authentication methods:

1. GitHub API Key: Set the `--gh-api-key` flag or configure the `repository.gh_api_token` setting in the configuration file.
2. SSH Key: Set the `--ssh-key` flag or configure the `repository.ssh_private_key` setting in the configuration file. The SSH key must be base64 encoded.

## Examples

Examples in YAML, JSON, and TOML:

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

If the Git blueprint fails, make these checks:

- Make sure that the repository URL is correct and accessible.
- Make sure that you gave the authentication details for private repositories.
- Make sure that the local path for the clone is valid and has the required permissions.

If the error persists, open an issue at [github.com/fynxlabs/rwr/issues](https://github.com/fynxlabs/rwr/issues).
