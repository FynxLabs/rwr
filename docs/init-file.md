# Init File - The Entrypoint

The Init file is the main entry point for your RWR blueprints. It defines the configuration settings for RWR and specifies the order of execution for the blueprints. This page describes the structure and options available in the Init file.

## File Format

The Init file is typically named `init.yaml`, but `init.yml`, `init.json` and
`init.toml` work as well. RWR supports YAML, JSON, and TOML formats for the Init
file.

## File Location

RWR finds the Init file in this order:

1. The path given to `--init-file` / `-i`.
2. The `repository.init-file` key in the [configuration file](cli/configuration.md).
3. `init.yaml`, `init.yml`, `init.json`, then `init.toml` in the current
   directory.

For 1 and 2 the path may be a **directory**, in which case RWR looks inside it
for those four names in that order. It may also be an `https://` URL, including a
GitHub `/blob/` URL, which RWR rewrites to the raw address. An `http://` URL is
refused: the Init file decides everything RWR runs, so it is not fetched in
cleartext.

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
| `schema_version` | The blueprint schema version for this tree. See [Schema versioning](schema-versioning.md) | No. The default is the latest version |

> [!NOTE]
> There is no `templatesEnabled` option. Templates are always processed.

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

The `variables` section holds the custom variables that your blueprints can read.

| Field | Description | Required |
|-------|-------------|----------|
| `userDefined` | A map of your own variables. Each key becomes `{{ .UserDefined.<key> }}` in a blueprint. The values may be strings, numbers, lists or nested maps | No |

```yaml
variables:
  userDefined:
    app_version: 1.0.0
    editors:
      - vim
      - neovim
```

`userDefined` is the only field you write here. The `{{ .User }}`,
`{{ .System }}` and `{{ .Flags }}` groups are also available to blueprints, but
RWR fills them in from the machine and from the flags you passed; they cannot be
set from the init file, so that a blueprint cannot claim to be running as a
different user or with different flags than it is. See
[Variables and Templating](variables.md).

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

In this example, the Init file specifies the format and location of the blueprint
files and the order of execution. It also configures package managers,
repositories, and defines custom variables. A blueprint in this tree reads them
as `{{ .UserDefined.app_version }}` and `{{ .UserDefined.api_key }}`.

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
