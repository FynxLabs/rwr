# Packages Blueprint

The Packages Blueprint allows you to manage packages on your system using RWR. You can specify packages to be installed or removed using various package managers, and now you can also provide additional arguments for package installation.

## Blueprint Structure

The Packages Blueprint has the following structure:

```yaml
packages:
  # Single package with base installation (no profiles)
  - name: package1
    action: install
    package_manager: apt
    elevated: true
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
    elevated: false
    args:
      - "--params"
      - "'/NoDesktopShortcut'"
```

## Blueprint Settings

The Packages Blueprint supports the following settings:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `names` or `import` is not provided | The name of the package to manage |
| `names` | Yes, if `name` or `import` is not provided | A list of package names to manage |
| `import` | Yes, if `name` or `names` is not provided | Path to import package definitions from another file (relative to blueprint directory) |
| `action` | Yes | The action to perform on the package(s) (`install` or `remove`) |
| `package_manager` | No | The package manager to use (e.g., `apt`, `brew`, `chocolatey`) |
| `elevated` | No | Whether to run the package manager with elevated privileges (default: `false`) |
| `args` | No | Additional arguments to pass to the package manager (as a list of strings) |
| `profiles` | No | List of profiles this package belongs to. If empty, package is always installed (base item) |

Note that you must provide either `name`, `names`, or `import` for each package entry. If both are provided, `names` will take precedence.

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

- Paths are resolved relative to your blueprint directory
- Prevents circular imports automatically
- Works with all package managers and formats
- Imported packages respect profile filtering
- Multiple imports can be used in a single file

For complete import examples, see [`examples/imports/`](../../examples/imports/).

## Supported Package Managers

RWR supports the following package managers out of the box:

- `apt` (Linux)
- `brew` (macOS, Linux)
- `chocolatey` (Windows)
- `dnf` (Linux)
- `yum` (Linux)
- `pacman` (Linux)
- `zypper` (Linux)
- `emerge` (Linux)
- `nix` (Linux, macOS)
- `scoop` (Windows)
- `winget` (Windows)

If a package manager is not specified, RWR will attempt to use the default package manager for the current operating system.

## Examples

Here are some examples of using the Packages Blueprint in different formats:

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
    elevated: true
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
      "elevated": true,
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
elevated = true
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

These examples demonstrate how to specify packages to be installed using different package managers and formats, including additional arguments for package installation.

## Additional Notes

- If a package is already installed, RWR will skip the installation process for that package.
- When removing packages, RWR will ignore any packages that are not currently installed.
- If a package manager is not available on the system, RWR will skip the package management process and log a warning.
- The `args` field allows you to pass additional arguments to the package manager. This is particularly useful for package managers like Homebrew (with `--cask`), Chocolatey (with installation parameters), or apt (with `--no-install-recommends`).

For more information on using the Packages Blueprint in your RWR configuration, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Commands and Flags](../cli/command-and-flags.md) pages.
