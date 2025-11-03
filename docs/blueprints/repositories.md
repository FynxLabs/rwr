# Repositories Blueprint

The Repositories Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage repositories for various package managers. You can add or remove repositories based on your system's requirements.

## Blueprint Structure

The Repositories Blueprint has the following structure:

```yaml
repositories:
  - name: string
    package_manager: string
    action: string
    url: string
    key_url: string (optional)
    channel: string (optional)
    component: string (optional)
    repository: string (optional)
```

## Blueprint Settings

The following settings are available for each repository in the Repositories Blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the repository |
| `import` | Yes, if `name` is not provided | Path to import repository definitions from another file (relative to blueprint directory) |
| `package_manager` | Yes | The package manager associated with the repository (e.g., apt, brew, dnf, zypper, pacman, choco, scoop) |
| `action` | Yes | The action to perform on the repository (`add` or `remove`) |
| `url` | Yes | The URL of the repository |
| `key_url` | No | The URL of the repository's GPG key (required for some package managers) |
| `channel` | No | The channel of the repository (applicable for some package managers) |
| `component` | No | The component of the repository (applicable for some package managers) |
| `repository` | No | The name of the repository (applicable for some package managers) |
| `profiles` | No | List of profiles this repository belongs to. If empty, repository is always managed (base item) |

## Blueprint Imports

Import repository definitions from other files:

```yaml
repositories:
  # Import shared repositories
  - import: ../../Common/repositories/base-repos.yaml

  # Add environment-specific repositories
  - name: custom-repo
    package_manager: apt
    action: add
    url: https://custom.example.com/repo
    key_url: https://custom.example.com/gpg
    profiles:
      - production
```

This allows you to maintain common repository configurations separately from environment-specific ones.

## Examples

Here are some examples of using the Repositories Blueprint in different formats:

### YAML

```yaml
repositories:
  - name: example-repo
    package_manager: apt
    action: add
    url: https://example.com/repo
    key_url: https://example.com/repo/gpg
    component: main
  - name: another-repo
    package_manager: brew
    action: add
    url: https://another-example.com/repo
```

### JSON

```json
{
  "repositories": [
    {
      "name": "example-repo",
      "package_manager": "apt",
      "action": "add",
      "url": "https://example.com/repo",
      "key_url": "https://example.com/repo/gpg",
      "component": "main"
    },
    {
      "name": "another-repo",
      "package_manager": "brew",
      "action": "add",
      "url": "https://another-example.com/repo"
    }
  ]
}
```

### TOML

```toml
[[repositories]]
name = "example-repo"
package_manager = "apt"
action = "add"
url = "https://example.com/repo"
key_url = "https://example.com/repo/gpg"
component = "main"

[[repositories]]
name = "another-repo"
package_manager = "brew"
action = "add"
url = "https://another-example.com/repo"
```

## Notes

- The Repositories Blueprint is processed by the `rwr run repository` command.
- The available package managers and their specific settings may vary depending on the operating system.
- Make sure to provide the correct URLs and GPG key URLs (if required) for the repositories you want to add.
- Removing a repository will not automatically remove the packages installed from that repository. You may need to manually remove them using the [Packages Blueprint](packages.md).
