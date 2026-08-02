# Packages Blueprint

With the Packages Blueprint, you manage packages on your system with RWR:

- install or remove packages with many package managers
- give additional arguments for the installation

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

The Packages Blueprint has the following structure:

```yaml
packages:
  # Single package with base installation (no profiles)
  - name: package1
    action: install
    package_manager: apt
    args:
      - "--no-install-recommends"

  # Single package removal
  - name: package2
    action: remove
    package_manager: brew

  # Multiple packages with profiles
  - names:
      - package3
      - package4
    action: install
    package_manager: chocolatey
    profiles:
      - work
    args:
      - "--params"
      - "'/NoDesktopShortcut'"
```

## Blueprint Settings

The Packages Blueprint supports the following settings:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `names` or `import` is not provided | The name of the package to manage. It must **not** begin with `-` |
| `names` | Yes, if `name` or `import` is not provided | A list of package names to manage. The same restriction applies to each |
| `import` | Yes, if `name` or `names` is not provided | Path to import package definitions from another file (relative to blueprint directory) |
| `action` | Yes | `install` or `remove`. See [Actions](#actions) |
| `package_manager` | No | The package manager to use (for example, `apt`, `brew`, or `chocolatey`) |
| `elevated` | No | Accepted by the schema but **not read**. Whether the package manager is run with elevation comes from the provider definition, not from the entry |
| `args` | No | Additional arguments to pass to the package manager (as a list of strings), appended after the package name |
| `profiles` | No | List of profiles this package belongs to. If empty, package is always installed (base item) |
| `interactive` | No | Override global interactive mode for this package (`true`/`false`). If omitted, uses the global `--interactive` flag |

Note that you must provide either `name`, `names`, or `import` for each package entry. If you give both `name` and `names`, `name` wins and RWR ignores `names`.

### Package names must not begin with `-`

- Every package manager reads a name that starts with `-` as an **option**
  (`--allow-downgrades`, `-U <url>`), not as a package.
- RWR refuses such a name and records it as a failure for that entry. The rest
  of the run continues.
- RWR runs commands as argv, not through a shell, so this is not shell
  injection. But such a name lets a blueprint change what the elevated package
  manager does.

### Actions

`install` and `remove` are implemented.

`update` is not:

- `rwr validate` accepts it, but the packages processor does **not implement
  it**.
- RWR records an entry that declares it as a failure with "unknown action" at
  run time.
- To refresh package lists, run a repositories blueprint — RWR runs each
  available provider's update command after processing it.

## Blueprint Imports

You can import package definitions from other files to share common package lists across multiple configurations:

```yaml
packages:
  # Import shared base packages
  - import: ../../Common/packages/base-packages.yaml

  # Import development tools
  - import: ../shared/dev-tools.yaml

  # Add system-specific packages
  - names:
      - system-specific-tool
      - custom-package
    action: install
    package_manager: apt
```

Import features:

- RWR resolves paths relative to your blueprint directory
- RWR prevents circular imports automatically
- Imports work with all package managers and formats
- Imported packages respect profile filtering
- You can use multiple imports in a single file

For complete import examples, see [`examples/imports/`](../../examples/imports/).

## Supported Package Managers

RWR ships provider definitions for:

- Linux: `apt`, `dnf`, `zypper`, `pacman`, `apk`, `xbps`, `eopkg`, `emerge`, `slackpkg`, the AUR helpers `yay`, `paru`, `aura`, `pamac` and `trizen`, plus `flatpak`, `snap` and `gnome-extensions`
- macOS: `brew`, `macports`, `mas`
- Windows: `chocolatey`, `scoop`, `winget`
- Cross-platform: `nix`, `cargo`

There is no `yum` provider. Use `dnf`.

When `package_manager` is omitted:

- RWR uses the default detected for this machine (which honors
  `/etc/os-release` and, on Arch, the AUR helper preference order).
- If that provider is not available, RWR uses the first available provider in
  alphabetical order.
- An unqualified package therefore gets the same manager on each run.

A named package manager that is not available on the machine logs a warning
and skips the entry.

## Examples

Examples in YAML, JSON, and TOML:

### YAML

```yaml
packages:
  # Base packages - always installed (no profiles field)
  - names:
      - git
      - curl
      - vim
    action: install
    package_manager: apt
    args:
      - "--no-install-recommends"

  # Development profile packages
  - names:
      - nodejs
      - python3
      - docker
    profiles:
      - dev
    action: install
    package_manager: apt

  # Work profile packages with multiple package managers
  - names:
      - visual-studio-code
      - google-chrome
      - brave-browser
    profiles:
      - work
    action: install
    package_manager: brew
    args:
      - "--cask"
```

### JSON

```json
{
  "packages": [
    {
      "names": [
        "git",
        "curl",
        "vim"
      ],
      "action": "install",
      "package_manager": "apt",
      "args": [
        "--no-install-recommends"
      ]
    },
    {
      "names": [
        "nodejs",
        "python3",
        "docker"
      ],
      "profiles": ["dev"],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "visual-studio-code",
        "google-chrome"
      ],
      "profiles": ["work"],
      "action": "install",
      "package_manager": "brew",
      "args": ["--cask"]
    }
  ]
}
```

### TOML

```toml
# Base packages - always installed (no profiles field)
[[packages]]
names = ["git", "curl", "vim"]
action = "install"
package_manager = "apt"
args = ["--no-install-recommends"]

# Development profile packages
[[packages]]
names = ["nodejs", "python3", "docker"]
profiles = ["dev"]
action = "install"
package_manager = "apt"

# Work profile packages
[[packages]]
names = ["visual-studio-code", "google-chrome", "brave-browser"]
profiles = ["work"]
action = "install"
package_manager = "brew"
args = ["--cask"]
```

## Additional Notes

- RWR does not check whether a package is already installed. It runs the provider's install command and lets the package manager decide. Every shipped provider's install command is idempotent for an already-installed package.
- A package that fails to install or remove does not stop the run. RWR collects the failures and reports them at the end.
- If a package manager is not available on the system, RWR skips that entry and logs a warning.
- With the `args` field, you pass additional arguments to the package manager. Use it for Homebrew (`--cask`), Chocolatey (installation parameters), or apt (`--no-install-recommends`).

For more information, see the [Blueprints Overview](../blueprints-general.md) and the [Commands and Flags](../cli/command-and-flags.md) pages.
