# Init File - The Entrypoint

The Init file is the main entry point for your RWR blueprints. It defines the configuration settings for RWR and specifies the order of execution for the blueprints. This page describes the structure and options available in the Init file.

## File Format

The Init file is typically named `init.yaml`, but you can also use `init.json` or `init.toml` depending on your preferred configuration format. RWR supports YAML, JSON, and TOML formats for the Init file.

## File Location

By default, RWR looks for the Init file in the current working directory. You can specify a different location using the `--init-file` or `-i` flag when running RWR commands.

## Init File Structure

The Init file consists of the following main sections:

### `blueprints`

The `blueprints` section defines the configuration settings for your blueprints.

| Field | Description | Required |
|-------|-------------|----------|
| `format` | The format of the blueprint files (yaml, json, toml) | Yes |
| `location` | The directory where the blueprint files are located | No (default: current directory) |
| `order` | The order of the blueprint types | No. The default is a fixed order. See [Blueprints General](blueprints-general.md) |
| `git` | Git repository settings for managing blueprints | No |
| `runOnlyListed` | Whether to run only the blueprints listed in the `order` field | No (default: false) |
| `templatesEnabled` | Whether to process template files | No (default: false) |
| `schema_version` | The blueprint schema version for this tree. See [Schema versioning](schema-versioning.md) | No. The default is the latest version |

> [!IMPORTANT]
> Do not put `packageManagers` in the `order` field. RWR installs the package
> managers from the `packageManagers` section before it reads the blueprint
> types. A `packageManagers` entry in `order` gives an "Unknown processor"
> warning and does nothing.

### `exposeCredentials`

The `exposeCredentials` section gives the credentials that the blueprints in this
tree can read. RWR withholds all credentials by default.

```yaml
exposeCredentials:
  - gh_api_token
```

See [Credentials](credentials.md) for the full description.

### `packageManagers`

The `packageManagers` section defines the configuration settings for package managers.

| Field | Description | Required |
|-------|-------------|----------|
| `name` | The name of the package manager | Yes |
| `action` | The action to perform (install, remove) | Yes |
| `asUser` | The user to run the package manager commands as | No |

### `repositories`

The `repositories` section defines the configuration settings for repositories.

| Field | Description | Required |
|-------|-------------|----------|
| `name` | The name of the repository | Yes |
| `package_manager` | The package manager associated with the repository | Yes |
| `action` | The action to perform (add, remove) | Yes |
| `url` | The URL of the repository | Yes |
| `key_url` | The URL of the repository's signing key | No |

### `variables`

The `variables` section allows you to define custom variables that can be used in your blueprints.

| Field | Description | Required |
|-------|-------------|----------|
| `user` | User-specific variables (username, home directory, etc.) | No |
| `flags` | Flag-specific variables (debug, log level, etc.) | No |
| `userDefined` | Custom variables defined by the user | No |

## Example Init File

Here's an example `init.yaml` file:

```yaml
blueprints:
  format: yaml
  location: blueprints
  order:
    - packages
    - repositories
    - files
  git:
    url: https://github.com/yourusername/rwr-blueprints.git
    branch: main
  runOnlyListed: true

packageManagers:
  - name: brew
    action: install

repositories:
  - name: homebrew-core
    package_manager: brew
    action: add
    url: https://github.com/Homebrew/homebrew-core.git

variables:
  userDefined:
    app_version: 1.0.0
    api_key: abc123
```

In this example, the Init file specifies the format and location of the blueprint files, the order of execution, and enables template processing. It also configures package managers, repositories, and defines custom variables.

### Package Manager Installation

The Init Process includes the ability to install and configure various package managers. This is particularly useful for setting up a consistent environment across different systems. Supported package managers include:

#### Homebrew (brew)

For macOS and Linux.

#### Nix (nix)

For macOS and Linux.

#### Chocolatey (chocolatey)

For Windows.

#### Scoop (scoop)

For Windows.

#### AUR Helpers

For Arch Linux:

- Yay (yay)
- Paru (paru)
- Trizen (trizen)
- Yaourt (yaourt)
- Pamac (pamac)
- Aura (aura)

#### Node.js Package Managers

- npm (npm)
- pnpm (pnpm)
- Yarn (yarn)

#### Pip (pip)

Python package manager.

#### RubyGems (gem)

Ruby package manager.

#### Cargo (cargo)

Rust package manager.

#### GNOME Extensions CLI (gnome-extensions)

For managing GNOME extensions.

Example configurations for package manager installation:

```yaml
packageManagers:
  - name: brew
    action: install
  - name: nix
    action: install
  - name: cargo
    action: install
    asUser: johndoe
  - name: gnome-extensions
    action: install
```

```json
{
  "packageManagers": [
    {
      "name": "brew",
      "action": "install"
    },
    {
      "name": "nix",
      "action": "install"
    },
    {
      "name": "cargo",
      "action": "install",
      "asUser": "johndoe"
    },
    {
      "name": "gnome-extensions",
      "action": "install"
    }
  ]
}
```

```toml
[[packageManagers]]
name = "brew"
action = "install"

[[packageManagers]]
name = "nix"
action = "install"

[[packageManagers]]
name = "cargo"
action = "install"
asUser = "johndoe"

[[packageManagers]]
name = "gnome-extensions"
action = "install"
```

These configurations can be included in your init file to ensure that the necessary package managers are installed during the initial setup process.

## Best Practices

- Keep your Init file concise and organized
- Use meaningful names for your variables
- Store sensitive information (e.g., API keys) in environment variables or secure vaults
- Use the `order` field to define the execution order of your blueprints explicitly

For more information on the specific blueprint types and their configuration options, please refer to the respective blueprint type documentation pages.
