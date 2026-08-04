# Package Manager Providers

The Providers system is a flexible and extensible way to manage package managers across different platforms. A provider is a declarative description of a package manager: the binary that identifies it, the files that prove it is in use, the command templates for install/remove/update, and the steps that add or remove a repository. Blueprints say `action: install`; the provider decides what that means on this machine.

## Where providers live

The providers shipped with RWR are authored in **CUE** under
`internal/system/definitions/` - the single source. The binary embeds
those files and evaluates them at load; the schema rejects an invalid
provider (wrong action name, missing `commands.install`, a `/tmp/` staging
path) naming the field. There is no exported or committed second
representation.

RWR loads provider overrides from the filesystem in three formats: **CUE**
(one provider document, validated against the same schema the embedded
definitions use), **JSON** (same shape), or **TOML** (the historical
`[provider]` layout). A filesystem provider replaces an embedded provider
of the same name.

### Search paths

RWR looks for a `providers/` directory in these locations, in order, and uses
the first one that exists:

1. Next to the `rwr` executable
2. `/usr/local/share/rwr/providers`
3. `/usr/share/rwr/providers`
4. `~/.config/rwr/providers`
5. macOS only: `/opt/homebrew/share/rwr/providers`,
   `/usr/local/Cellar/rwr/providers`, `/Applications/rwr/providers`

The current working directory is deliberately **not** searched. A provider file
declares `exec`, `args` and `elevated = true` and those run verbatim, so
honouring `./providers` would hand root-level execution to any directory rwr
happens to be run from - a cloned blueprint repo, `/tmp`, a shared downloads
folder.

For the same reason, a provider file (or its directory) that is group- or
world-writable is skipped with a warning; the rest of the directory still
loads.

## Provider Configuration

A filesystem override in TOML looks like this:

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
install = "install"   # Package installation command (required)
update = "update"    # System update command
remove = "remove"    # Package removal command
list = "list"       # List installed packages
search = "search"    # Search for packages
clean = "clean"     # Clean package cache
```

`install` is the only required command. A provider that declares no `clean`
command is simply not invoked during end-of-run cache cleaning.

`environment` is an optional table of environment variables set for every
package command the provider runs:

```toml
[provider.environment]
SOME_FLAG = "1"
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

Define packages required by the provider:

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
sha256 = "{{ .KeySha256 }}"
condition = "{{ .HasKey }}"

[[provider.repository.add.steps]]
action = "command"              # Import GPG key
exec = "gpg"
args = ["--import", "{{ .KeyPath }}"]
condition = "{{ .HasKey }}"
```

`remove` steps are declared the same way under
`[[provider.repository.remove.steps]]`. A `remove` step that deletes a file is
confined to the directories the provider declares as its repository paths.

### Installation Steps

Define how to install (or remove) the provider itself:

```toml
[[provider.install.steps]]
action = "download"             # Download an installer
source = "https://example.com/provider-install.sh"
dest = "{{ .TempDir }}/provider-install.sh"

[[provider.install.steps]]
action = "command"              # Run it
exec = "sh"
args = ["{{ .TempDir }}/provider-install.sh"]
```

`[[provider.remove.steps]]` uses the same three actions to uninstall the
provider.

Do not stage files at fixed `/tmp/` paths - any local user can pre-create or
rewrite such a path between the download and the elevated step that executes
it. `{{ .TempDir }}` renders to a per-run `0700` directory other users cannot
reach, and the CUE schema refuses to export a shipped provider whose install
steps mention `/tmp/`.

## Available Actions

Install and remove steps of a provider accept these actions:

- `command` - Run a command (`exec` + `args`)
- `download` - Get a file from a URL (`source` → `dest`, optional `sha256`)
- `write` - Write `content` to a file at `dest`

Repository `add`/`remove` steps accept these actions:

- `command` (or `exec`) - Run a command
- `download` - Get a file from a URL, verified against `sha256` when given
- `write` - Write `content` to a file at `dest`
- `copy` - Copy a file from `source` to `dest`
- `append` - Append `content` to the file at `path`
- `remove` - Delete the file at `path` (confined to the provider's repository
  paths)
- `remove_line` - Delete the lines matching `match` from the file at `path`
- `remove_section` - Delete the named `section` from the file at `path`

A step with any other action stops the run with an error, and `rwr validate
--providers` reports it beforehand.

### Step fields

Each step may carry: `action`, `exec`, `args`, `source`, `dest`, `content`,
`path`, `match`, `section`, `sha256` and `condition`.

`condition` is a template that gates the step: only a step whose condition
renders truthy runs. Conditions may reference the derived predicates -
`HasKey`, `HasInterfaces`, `HasSlot`, `HasProxy`, `HasToken`,
`HasAuthentication`, `RequiresAuth`, `IsCustomRegistry`, `IsOverlay`,
`IsMainRepo`, `IsLocalFile`, `IsLocalSnap`, `IsSnapStore`, `UserMode` - plus
the step-data fields `ResetSettings` and `URL`. A condition naming anything
else is an error at the point the step would have run.

## Template Variables

Every templated field of a step is rendered - `source`, `dest`, `exec`,
`content`, `path`, `match`, `section`, `sha256` and each entry of `args` - with
`missingkey=error`, so a placeholder rwr cannot fill is reported rather than
written to disk as literal text.

Repository steps render against the repository entry being processed:

- `{{ .Name }}` - Name of the repository (also used to derive
  `{{ .KeyPath }}` = `<keys dir>/<name>.gpg`)
- `{{ .URL }}` - URL of the repository
- `{{ .KeyURL }}` / `{{ .KeyID }}` / `{{ .KeySha256 }}` - The signing key: its
  URL, its ID, and the digest the key download is verified against
- `{{ .KeyPath }}` - Path of the key on the machine
- `{{ .TempKeyPath }}` - Per-run private staging path for the key
- `{{ .SourcesPath }}` / `{{ .KeysPath }}` / `{{ .ConfigPath }}` - The
  provider's declared repository paths
- `{{ .Arch }}`, `{{ .Channel }}`, `{{ .Component }}`, `{{ .Repository }}`,
  `{{ .Description }}` - Fields written into source lines and `.repo` files
- `{{ .PackageManager }}`, `{{ .Action }}` - The entry's own manager and action
- `{{ .OverlayPath }}`, `{{ .SyncType }}`, `{{ .SHA256 }}` - Portage/nix
  overlay fields
- `{{ .ProxyURL }}` - Proxy snapd should route through
- `{{ .UUID }}`, `{{ .ExtensionID }}` - GNOME extension identifiers
- `{{ .Interface }}`, `{{ .Slot }}`, `{{ .ResetSettings }}` - Snap interface
  wiring and GNOME extension settings reset
- `{{ .Path }}` - The local file a sideloaded snap or extension is installed
  from
- `{{ .Username }}`, `{{ .Password }}`, `{{ .Token }}` - Credentials for a
  private source (kept out of rwr's own logs)

Install and remove steps render against a single variable: `{{ .TempDir }}`,
the run's private staging directory. Blueprint variables (`.UserDefined`,
`.System`, …) are **not** in scope inside provider steps - provider steps
describe the machine's package manager, not the blueprint.

## Supported Providers

These are the definitions shipped in the binary (one per file under
`providers/cue/`):

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

### Desktop Environment

- gnome-extensions - GNOME Shell extensions

There are no shipped definitions for npm, pnpm, yarn, pip or gem. To manage
packages from those ecosystems today, install them with a `scripts` blueprint
or add your own provider definition.

## Creating New Providers

To change a shipped provider or add support for a new package manager without
touching the source tree:

1. Copy an exported JSON definition from
   `internal/system/definitions/providers/` (or write a TOML file in the shape
   shown above) into one of the search paths - `~/.config/rwr/providers/` for
   a per-user override.
2. Make sure the file is not group- or world-writable, or it will be skipped.
3. Configure the provider sections: basic information, detection rules,
   commands, core packages, repository management, install/remove steps.
4. Check it with `rwr validate <file> --providers` and test with example
   blueprints.

To contribute a provider to RWR itself, author it in CUE under
`internal/system/definitions/` - the schema there documents every field
and rejects invalid definitions. That's it: no export step, nothing else to
regenerate.

## Best Practices

- Use `elevated = true` for system-wide package managers
- Include all relevant detection files
- Document command flags in comments
- Use consistent repository paths
- Break complex operations into clear steps
- Stage downloads under `{{ .TempDir }}`, never at fixed `/tmp` paths
- Pin downloaded content with `sha256` where the upstream publishes digests
- Validate repository configurations
- Test on supported platforms

## Distribution-Specific Alternatives

Some package managers are used across multiple distributions but may have different package names for the same functionality. RWR supports distribution-specific alternatives to handle these differences without requiring separate provider files.

### How Alternatives Work

The alternatives system allows a single provider to specify different package names for different distributions:

```toml
[provider.alternatives.distribution_name]
  [provider.alternatives.distribution_name.corePackages]
  openssl = ["alternative-openssl-package", "alternative-openssl-devel"]
  build-essentials = [
    "alternative-make",
    "alternative-cmake"
  ]
```

When RWR detects the specified distribution, it will use the alternative package names instead of the default ones.

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

This allows OpenMandriva users to use RWR with the DNF provider while automatically getting the correct package names for their distribution.

### Benefits

- Single provider file for related distributions
- Automatic package name resolution based on detected distribution
- Easy to extend for new distributions with naming variations
- Maintains backward compatibility

## Future Enhancements

- Support for more package managers
- Better dependency resolution
- Repository mirroring
- Version pinning
- Rollback support
- Plugin system
- Extended alternatives system for commands and repository paths
