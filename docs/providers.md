# Package Manager Providers

The Providers system manages package managers on different platforms. TOML files define how each package manager works. You can add a new package manager without Go code.

## Provider Configuration

You configure a provider with a TOML file in the `internal/system/definitions/providers/` directory. For example, `internal/system/definitions/providers/apt.toml` defines the configuration for the APT package manager.

### Basic Structure

```toml
[provider]
name = "provider-name"  # Unique identifier
elevated = false       # Whether root/admin privileges are needed

[provider.detection]
binary = "binary-name" # Main executable to check for
files = [             # Files that indicate installation
    "/path/to/binary",
    "/path/to/config"
]
distributions = [      # Where this provider is available
    "debian",
    "ubuntu"
]

[provider.commands]
install = "install"   # Package installation command
update = "update"    # System update command
remove = "remove"    # Package removal command
list = "list"       # List installed packages
search = "search"    # Search for packages
clean = "clean"     # Clean package cache
```

## How RWR finds a provider

RWR makes the provider available when these two conditions are true:

1. The `binary` is on the system.
2. Each path in `files` is on the system.

The `distributions` list is a hint. A match in that list is not necessary.

This is intentional. There are many derivatives of Arch and Debian, and one list
cannot contain all of them. Some derivatives give a new value in
`/etc/os-release` and give no `ID_LIKE` value. A machine that has the `pacman`
binary and the `/var/lib/pacman` database uses pacman. The name of the
distribution does not change that fact.

The opposite condition is also correct. A machine with the name "Arch" that uses
apt has no `apt` binary and no `/etc/apt` directory. RWR does not make apt
available on that machine. RWR tests each package manager, not the name of the
distribution.

Give a `files` list for each provider. Without that list, RWR can find the
provider only from a match in `distributions`.

RWR also finds the distribution family from the installed package manager. This
gives the correct default package manager on a derivative that no list contains.

### Core Packages

Define the packages that the provider requires:

```toml
[provider.corePackages]
openssl = [           # SSL/TLS packages
    "openssl",
    "ca-certificates"
]
build-essentials = [  # Build tools
    "base-devel",
    "build-essential"
]
```

### Repository Management

Configure repository paths and management steps:

```toml
[provider.repository.paths]
sources = "/etc/apt/sources.list.d"  # Repo definitions
keys = "/etc/apt/trusted.gpg.d"      # GPG keys
config = "/etc/apt/apt.conf.d"       # Configuration

[[provider.repository.add.steps]]
action = "download"              # Download GPG key
source = "{{ .KeyURL }}"
dest = "{{ .KeyPath }}"

[[provider.repository.add.steps]]
action = "command"              # Import GPG key
exec = "gpg"
args = ["--import", "{{ .KeyPath }}"]
```

### Installation Steps

Define how to install the provider itself:

```toml
[[provider.install.steps]]
action = "command"              # Install dependencies
exec = "package-manager"
args = ["install", "dependency1"]

[[provider.install.steps]]
action = "mkdir"                # Create directories
path = "/path/to/create"
mode = "0755"

[[provider.install.steps]]
action = "download"             # Download provider
source = "https://example.com/provider.tar.gz"
dest = "/tmp/provider.tar.gz"
```

## Available Actions

RWR reads these actions in the installation steps of a provider:

- `command` - Run a command
- `download` - Get a file from a URL
- `write` - Write content to a file

RWR reads these actions in the repository steps of a provider:

- `command` - Run a command
- `write` - Write content to a file
- `copy` - Copy a file

CAUTION: Use only the actions in these two lists. RWR stops with an error when a
step gives a different action. Some provider files in this repository contain
other actions, and those steps do not operate.

## Template Variables

These variables are in the repository steps of the provider files:

- `{{ .Name }}` - Name of the repository or package
- `{{ .URL }}` - URL of the repository
- `{{ .KeyURL }}` - URL of the GPG key
- `{{ .KeyPath }}` - Path of the key on the machine
- `{{ .SourcesPath }}` - Path of the repository configuration
- `{{ .Arch }}` - Architecture of the system
- `{{ .Channel }}` - Channel of the repository
- `{{ .Component }}` - Component of the repository

CAUTION: RWR replaces only `{{ .URL }}`, and only in the `args` of a step. RWR
does not replace the other variables. RWR does not replace a variable in the
`dest` or `content` of a step. A step that uses one of those variables writes the
text of the variable to the file.

This is a known fault. Do not write a new provider that depends on these
variables until the fault is corrected.

## Supported Providers

### Linux Package Managers

- apt - Debian, Ubuntu
- dnf - Fedora, RHEL, OpenMandriva
- pacman - Arch Linux
- zypper - openSUSE
- apk - Alpine Linux
- emerge - Gentoo
- xbps - Void Linux
- eopkg - Solus
- slackpkg - Slackware

### Linux AUR Helpers

- paru - Arch User Repository
- yay - Arch User Repository
- trizen - Arch User Repository
- aura - Arch User Repository
- pamac - Arch User Repository

### Linux Universal Package Managers

- flatpak - Universal Linux packages
- snap - Universal Linux packages
- nix - Universal package manager

### macOS Package Managers

- brew - Homebrew
- macports - MacPorts
- mas - Mac App Store

### Windows Package Managers

- chocolatey - Windows package manager
- winget - Windows Package Manager
- scoop - Windows package manager

### Language Package Managers

- cargo - Rust packages
- npm/pnpm/yarn - Node.js packages
- pip - Python packages
- gem - Ruby packages

### Desktop Environment

- gnome-extensions - GNOME Shell extensions

## Creating New Providers

To add support for a new package manager:

1. Copy `internal/system/definitions/provider_template.toml` to `internal/system/definitions/providers/<name>.toml`
2. Configure the provider sections:
   - Basic information (name, elevation)
   - Detection rules (binary, files, distributions)
   - Standard commands
   - Core package requirements
   - Repository management
   - Installation/removal steps
3. Test the provider with example blueprints

## Best Practices

- Use `elevated = true` for system-wide package managers
- Include all relevant detection files
- Document command flags in comments
- Use consistent repository paths
- Break complex operations into clear steps
- Validate repository configurations
- Stop with a clear error message when a step fails
- Test on supported platforms

## Distribution-Specific Alternatives

Multiple distributions use the same package manager, but the package names can be different. Alternatives handle these differences. You do not need a separate provider file for each distribution.

### How Alternatives Work

With alternatives, one provider gives different package names for different distributions:

```toml
[provider.alternatives.distribution_name]
  [provider.alternatives.distribution_name.corePackages]
  openssl = ["alternative-openssl-package", "alternative-openssl-devel"]
  build-essentials = [
    "alternative-make",
    "alternative-cmake"
  ]
```

When RWR detects the specified distribution, it uses the alternative package names instead of the default ones.

### Example: OpenMandriva Support

OpenMandriva uses DNF as its package manager but has different package naming conventions. The DNF provider includes alternatives for OpenMandriva:

```toml
[provider.alternatives.openmandriva]
  [provider.alternatives.openmandriva.corePackages]
  openssl = ["openssl", "lib64openssl-devel"]
  build-essentials = [
    "make",
    "cmake",
    "lib64freetype6-devel",
    "lib64fontconfig-devel",
    "lib64xcb-devel",
    "lib64xkbcommon-devel",
    "gcc-c++"
  ]
```

OpenMandriva users can use RWR with the DNF provider. RWR selects the correct package names automatically.

### Benefits

- One provider file serves related distributions.
- RWR selects the package names for the detected distribution.
- You can add a distribution with different names.
- Existing provider files continue to work.

## Future Enhancements

- Support for more package managers
- Better dependency resolution
- Package verification/signing
- Repository mirroring
- Version pinning
- Rollback support
- Plugin system
- Extended alternatives system for commands and repository paths
