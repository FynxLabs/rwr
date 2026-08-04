# Rinse, Wash, Repeat (RWR)<!-- omit in toc -->

[![Build](https://github.com/FynxLabs/rwr/actions/workflows/go.yml/badge.svg)](https://github.com/FynxLabs/rwr/actions/workflows/go.yml)
[![Coverage gated ≥50%](https://img.shields.io/badge/coverage-gated%20%E2%89%A550%25-blue)](https://github.com/FynxLabs/rwr/blob/master/scripts/coverage-gate.sh)

![RWR Logo](img/rwr_128.gif)

Rinse, Wash, Repeat (RWR) is a powerful and flexible configuration management tool designed for those who like to hop around and reinstall frequently, regardless of whether it's Linux, macOS, or Windows. It aims to simplify the process of setting up and maintaining your system, making it easy to rebuild and reproduce configurations across multiple machines - and, with its run records, to see what a run changed and to reverse it again.

## Features

- **Blueprint-based Configuration**: Uses configuration files called blueprints to define and manage your system's configuration
- **Blueprint Imports**: Share and reuse blueprint configurations across multiple files and projects
- **Profile System**: Additive profile model for managing different environments (dev, staging, production) or use cases (work, personal)
- **Multi-format Support**: Blueprints can be written in YAML, JSON, TOML, or CUE format, and `rwr convert` rewrites a whole tree from one format to another
- **Multi-configuration Repos**: One blueprint repository can hold several configurations behind a root manifest; RWR matches the machine, or `--config-name` picks one
- **Task-runner CLI**: `rwr all` runs everything; `rwr run <processor>` (or the shorthand `rwr packages`) runs one processor; `rwr bootstrap` runs just the bootstrap step
- **Interactive Dashboard**: Interactive runs get a live terminal dashboard with built-in and user-defined themes; non-interactive runs keep plain streaming logs
- **Run Records**: Every run writes a journal; `rwr status` shows desired-vs-actual drift and `rwr uninstall` reverses what recorded runs applied
- **Managed Credentials**: Declare credentials in the init file and source them from environment variables, the OS keyring, or a prompt - redacted in logs by default
- **Cross-platform Package Management**: Integrates with various package managers across Linux, macOS, and Windows
- **File & Directory Management**: Copy, move, delete, create, and manage permissions with URL source support
- **Service Management**: Start, stop, enable, and disable system services
- **Repository Management**: Manage package repositories for apt, brew, dnf, zypper, and more
- **User & Group Management**: Create and manage user accounts and groups
- **Template Rendering**: Dynamic configurations with variable substitution
- **Git Repository Management**: Clone and manage Git repositories
- **Script Execution**: Execute scripts with multiple interpreter support
- **SSH Key Management**: Generate and manage SSH keys with GitHub integration
- **Extensible Architecture**: Package managers are declarative providers - authored in CUE and embedded in the binary, with filesystem overrides in TOML or JSON

## Table of Contents<!-- omit in toc -->

- [Features](#features)
- [Quick Install](#quick-install)
  - [Unix-based Systems (Linux and macOS)](#unix-based-systems-linux-and-macos)
  - [Windows](#windows)
- [Installation](#installation)
  - [Packages](#packages)
  - [From Releases](#from-releases)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Profile System](#profile-system)
- [Blueprint Imports](#blueprint-imports)
- [Blueprint Types](#blueprint-types)
- [Documentation](#documentation)
  - [Documentation Index](#documentation-index)
  - [RWR Command Line Interface](#rwr-command-line-interface)
  - [Blueprints](#blueprints)
  - [Advanced Topics](#advanced-topics)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Setting Up Development Environment](#setting-up-development-environment)
  - [Development Commands](#development-commands)
  - [CI/CD Pipeline](#cicd-pipeline)
    - [Environment Variables](#environment-variables)
- [Contributing](#contributing)
  - [Specs](#specs)
- [License](#license)
- [Contact](#contact)

## Quick Install

For a quick installation of RWR, you can use the following one-liners:

### Unix-based Systems (Linux and macOS)

```bash
curl -sSL https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.sh | sudo bash
```

### Windows

Open PowerShell as an administrator and run:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.ps1'))
```

Each script picks the build for your machine from the latest release, verifies it
against the release's `checksums.txt` and refuses to install on a mismatch.
`install.sh` installs to `/usr/local/bin`; `install.ps1` installs to
`%ProgramFiles%\rwr` and adds it to the machine `PATH`.

To install the rolling [nightly prerelease](https://github.com/FynxLabs/rwr/releases/tag/nightly)
(an unvetted build of master) or pin a specific version, pass a flag through:

```bash
curl -sSL https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.sh | sudo bash -s -- --nightly
```

`--tag v0.5.1` pins a release by tag. On Windows the same options are
`-Nightly` and `-Tag v0.5.1` (download the script to a file to pass
parameters, or wrap the one-liner in `& ([scriptblock]::Create(...))`).

> [!NOTE]
> Always review scripts before running them with elevated privileges. The install scripts are available for inspection in the RWR repository.

## Installation

### Packages

Get the packages from the
[releases page](https://github.com/fynxlabs/rwr/releases). These formats are
available:

- Archives (`.tar.gz` and `.zip`)
- Debian packages (`.deb`)
- RPM packages (`.rpm`)
- Alpine packages (`.apk`)
- Arch packages (`.pkg.tar.zst`)
- A Homebrew tap

RWR builds for Linux, macOS and Windows on `x86_64` and `arm64`, for Linux on
`armv7`, and for Linux on `riscv64`.

### Builds of the master branch

Each merge to `master` publishes a prerelease with the tag `nightly`.

WARNING: A `nightly` build is not a release. It is the master branch at the time
of the build. Use the
[latest release](https://github.com/fynxlabs/rwr/releases/latest) for a machine
that you depend on.

### From Releases

To install RWR from a release:

1. Get the archive for your machine from the
   [releases page](https://github.com/fynxlabs/rwr/releases).
2. Compare the file against `checksums.txt` from the same release.
3. Extract the archive.
4. Add the directory to your `PATH`.

Read [Install](docs/install.md) for the full description.

## Getting Started

1. **Make the configuration file**: Run
   [`rwr config create`](docs/cli/configuration.md). RWR asks for the settings
   and writes the file.
2. **Give the blueprints**: Give a git repository URL, a local path, or a
   GitHub shorthand like `owner/repo` when RWR asks for it.
3. **Set up the system**: Run [`rwr all`](docs/cli/command-and-flags.md). RWR
   applies the blueprints.

For detailed setup instructions, see the [Quick Start Guide](docs/quick-start.md).

## Configuration

RWR's configuration file (`~/.config/rwr/config.yaml` by default) holds where to
find the init file and the credentials used to reach a private one:

```yaml
repository:
  init-file: https://github.com/your-org/your-blueprints/blob/main/init.yaml
  gh_api_token: your_github_api_token
  ssh_private_key: your_ssh_private_key_base64

log:
  level: info
```

`init-file` may be a local path, a directory to look in, or an `https://` URL.
Where the blueprints themselves come from - a git remote or a local directory -
is decided by the init file, under `blueprints.location` and `blueprints.git`.

Point RWR at a different configuration with `--config`, which takes either a file
or a directory holding `config.yaml`.

See the [Configuration documentation](docs/cli/configuration.md) for complete setup details.

## Profile System

RWR uses an additive profile system where:

- **Base items** (no profiles specified) are always applied
- **Profile items** are only applied when their profiles are active
- Multiple profiles can be activated simultaneously

```bash
# Apply everything: with no profile named, nothing is filtered out
rwr all

# Apply base + dev profile
rwr all --profile dev

# Apply base + multiple profiles
rwr all --profile dev --profile work
```

> [!NOTE]
> The profile filter runs only when you name at least one profile. `rwr all` on
> its own applies profile items too.

For detailed information, see the [Profile System documentation](docs/profiles.md).

## Blueprint Imports

RWR supports importing blueprint definitions from other files, enabling you to share common configurations across multiple systems:

```yaml
packages:
  # Import shared packages from another file
  - import: ../../Common/packages/base.yaml
  # Add system-specific packages
  - names:
      - custom-package
    action: install
```

Import features:

- **All blueprint types**: packages, files, services, git, scripts, and the
  others
- **Profile support**: RWR applies the profile filter to the items that it
  imports

> [!NOTE]
> An import brings in one file. RWR does not read the imports inside the file
> that it imports. Put each import in the file that needs it.
>
> Most blueprint types resolve an import path against the `location` in the init
> file. Files, templates, directories and scripts resolve the path against the
> directory of the blueprint file. Use a path that is correct for the type.

See the [examples/imports/](examples/imports/) directory for detailed examples.

## Blueprint Types

RWR supports these blueprint types:

- **packages** - Install and remove packages with a package manager
- **repositories** - Manage the package repositories
- **files** - Copy, move, delete, and link files. The `directories` and
  `templates` keys are part of this type
- **services** - Manage the system services
- **configuration** - Set the desktop and system settings
- **git** - Clone and update git repositories
- **scripts** - Run scripts with a program that you select
- **users** - Manage the user accounts and the groups
- **ssh_keys** - Make SSH keys and send them to GitHub
- **fonts** - Install fonts
- **bootstrap** - Prepare the system before the other types run

For detailed blueprint documentation, see the [Blueprint Types](docs/index.md#blueprints) section.

## Documentation

For detailed documentation on how to use RWR, please refer to the `docs/` directory. Here's an overview of the topics covered:

### Documentation Index

- [Documentation Index](docs/index.md)
- [Install](docs/install.md)
- [Quick Start Guide](docs/quick-start.md)
- [What are Blueprints?](docs/blueprints-general.md)
- [Init File - The Entrypoint](docs/init-file.md)
- [Bootstrap - System Prerequisites](docs/bootstrap.md)

### RWR Command Line Interface

- [CLI Commands & Flags](docs/cli/command-and-flags.md)
- [Config File](docs/cli/configuration.md)
- [Profile CLI Commands](docs/cli/profiles.md)
- [Validate Command](docs/cli/validate.md)
- [Convert Command](docs/cli/convert.md)

### Blueprints

- [Blueprint Best Practices](docs/best-practices.md)
- [Fields Common to Every Blueprint](docs/blueprints/common-fields.md)
- Blueprint Types
  - [Packages Blueprint](docs/blueprints/packages.md)
  - [Repositories Blueprint](docs/blueprints/repositories.md)
  - [Configuration Blueprint](docs/blueprints/configuration.md)
  - [Files Blueprint](docs/blueprints/files.md)
  - [Directories Blueprint](docs/blueprints/directories.md)
  - [Fonts Blueprint](docs/blueprints/fonts.md)
  - [Services Blueprint](docs/blueprints/services.md)
  - [Users and Groups Blueprint](docs/blueprints/users-and-groups.md)
  - [Git Blueprint](docs/blueprints/git.md)
  - [Scripts Blueprint](docs/blueprints/scripts.md)
  - [SSH Keys Blueprint](docs/blueprints/ssh-keys.md)

### Advanced Topics

- [Profile System - Environment & Use Case Management](docs/profiles.md)
- [Profile Best Practices](docs/profile-best-practices.md)
- [Template Variables](docs/variables.md)
- [Package Manager Providers](docs/providers.md)
- [Credentials](docs/credentials.md)
- [Run Records - Status & Uninstall](docs/state.md)
- [Schema versioning](docs/schema-versioning.md)

For more detailed information on each topic, please refer to the corresponding documentation file.

## Development

### Prerequisites

RWR uses [mise](https://mise.jdx.dev/) to manage development tools. Install mise following their documentation.

### Setting Up Development Environment

1. Clone the repository:

    ```bash
    git clone https://github.com/fynxlabs/rwr.git
    cd rwr
    ```

2. Install the tools:

    ```bash
    mise install
    ```

    This installs Go, GoReleaser, gotestsum, golangci-lint, Node, pkl, hk and
    the OpenSpec CLI.

    `mise install` also runs a postinstall hook that does the rest of the setup:
    it installs the git hooks defined in `hk.pkl` (`hk install --mise`, which
    gives you the pre-commit and pre-push checks), initializes or updates the
    OpenSpec scaffold under `openspec/`, runs `go mod download`, and installs
    `govulncheck` if it is missing. Run it once after cloning and the tree is
    ready to build.

### Development Commands

Build and run:

```bash
mise run build          # Build the binary
mise run start -- all   # Run rwr locally; put the arguments after --
mise run install        # Build and copy to ~/.local/bin (Linux and macOS)
```

Test:

```bash
mise run test           # All tests, with gotestsum formatting
mise run test:unit      # The internal packages only
mise run test:coverage  # Tests with a coverage report
mise run test:watch     # Run the tests again at each file change
```

To test one package:

```bash
mise run test:helpers
mise run test:processors
mise run test:system
```

Each test task has a `:raw` form that uses `go test` without the gotestsum
formatting. Examples are `mise run test:raw` and `mise run test:unit:raw`.

Check the code:

```bash
mise run lint       # golangci-lint
mise run security   # govulncheck
mise run format     # go fmt
```

Run the full check before you push:

```bash
mise run ci         # test, lint, and security together
```

`mise run ci` is not the same set of checks as the CI workflow: it runs the tests
without `-race`, on this machine only, and it does not run gosec, the installer
checks, or the example validation. Read the [CI/CD Pipeline](#cicd-pipeline)
section for what the workflow actually does. The git hooks installed by
`mise install` cover part of the gap: `pre-commit` runs golangci-lint with
`--fix`, and `pre-push` runs `go test -race ./...` and `govulncheck`.

Update the dependencies:

```bash
mise run update     # go get -u ./... and go mod tidy
```

### CI/CD Pipeline

GitHub Actions runs the pipeline. There are three workflows.

`Go Build & Test` runs at each push to `master` and at each pull request -
every pull request, not only those targeting `master`, so a branch stacked on
another branch is still checked. Its jobs:

- `build` runs on `ubuntu-latest`, `macos-latest` and `windows-latest`. Each
  runs `go build ./...` and `go vet ./...`; vet type-checks the test files, so
  this is also the guarantee that the whole tree compiles on that platform.
  `go test -race -v ./...` gates on Linux, and a coverage gate keeps total
  coverage at or above a ratcheting threshold. The tests also run on macOS and
  Windows, but with `continue-on-error`, because much of the suite still
  assumes POSIX behaviour.
- `cross-compile` builds every release target with `goreleaser build
  --snapshot`, and `release-snapshot` runs the full release pipeline - signing
  included - as a snapshot, so a release-only breakage is caught before merge.
- `cue-providers` vets the CUE provider sources under `providers/cue/` and
  fails when the committed JSON under
  `internal/system/definitions/providers/` differs from a fresh export.
- `lint` runs golangci-lint against the repository's own `.golangci.yml`, so it
  is the same set of rules as `mise run lint`.
- `installers` runs shellcheck over `install.sh` and parses `install.ps1` with
  the PowerShell parser. Both gate. PSScriptAnalyzer also runs, reporting only
  for now. The scripts are analysed, never executed.
- `examples` checks each file in `examples/`: the file parses in its format, the
  template variables exist, the fields decode strictly into the blueprint structs
  with no unknown keys, and the YAML, JSON and TOML copies describe the same
  thing. It then builds the binary and runs `rwr validate` over every example
  tree, and asserts that the set of example directories `rwr validate` cannot
  reach has not grown.
- `security` runs gosec and govulncheck.

`RWR - Master Prerelease` runs at each merge to `master`, and can be started by
hand. It builds the branch and publishes the files under the `nightly` tag. Read
[Install](docs/install.md) for more information.

`RWR - Release` runs at each pushed tag. It builds the binaries and the packages,
creates the GitHub release, and updates the Homebrew tap.

#### Environment Variables

The release workflow needs these secrets:

- `GITHUB_TOKEN` - creates the release and sends the files to it.
- `HOMEBREW_TAP_DEPLOY_KEY` - an SSH key that can write to the Homebrew tap
  repository.
- `NAUR_DISPATCH_TOKEN` - starts the update of the naur repository.

## Contributing

Contributions to RWR are welcome! If you'd like to contribute, please follow these steps:

1. Fork the repository on GitHub.
2. Create a new branch for your feature or bug fix.
3. Make your changes and commit them with descriptive commit messages.
4. Push your changes to your forked repository.
5. Submit a pull request to the main repository.

Please ensure that your code follows the project's coding style and includes appropriate tests.

### Specs

RWR keeps living specifications under `openspec/specs/`, one directory per
capability: `blueprint-processing`, `blueprint-schema-versioning`,
`blueprint-validation`, `cli`, `command-execution`, `credential-handling`,
`distribution`, `initialization`, `provider-detection` and `state-tracking`.
Project-wide context
and the rules specs are written against live in `openspec/config.yaml`.

**If a change alters behavior that a spec covers, update that spec in the same
pull request.** A spec that describes behavior the code no longer has is worse
than no spec. If a requirement is not implemented yet, record it under a "Known
Gaps" heading rather than writing intent as if it shipped.

`mise install` sets up the OpenSpec tooling; `openspec --help` lists what it can
do.

## License

RWR is open-source software licensed under the [MIT License](LICENSE).

## Contact

If you have any questions, suggestions, or feedback, please open an issue on the [GitHub repository](https://github.com/fynxlabs/rwr/issues) or contact the maintainers directly.

Happy distrohopping with RWR!
