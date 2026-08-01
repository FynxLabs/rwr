# Rinse, Wash, Repeat (RWR)<!-- omit in toc -->

![RWR Logo](img/rwr_128.gif)

Rinse, Wash, Repeat (RWR) is a powerful and flexible configuration management tool designed for those who like to hop around and reinstall frequently, regardless of whether it's Linux, macOS, or Windows. It aims to simplify the process of setting up and maintaining your system, making it easy to rebuild and reproduce configurations across multiple machines.

## Features

- **Blueprint-based Configuration**: Uses configuration files called blueprints to define and manage your system's configuration
- **Blueprint Imports**: Share and reuse blueprint configurations across multiple files and projects
- **Profile System**: Additive profile model for managing different environments (dev, staging, production) or use cases (work, personal)
- **Multi-format Support**: Blueprints can be written in YAML, JSON, or TOML format
- **Cross-platform Package Management**: Integrates with various package managers across Linux, macOS, and Windows
- **File & Directory Management**: Copy, move, delete, create, and manage permissions with URL source support
- **Service Management**: Start, stop, enable, and disable system services
- **Repository Management**: Manage package repositories for apt, brew, dnf, zypper, and more
- **User & Group Management**: Create and manage user accounts and groups
- **Template Rendering**: Dynamic configurations with variable substitution
- **Git Repository Management**: Clone and manage Git repositories
- **Script Execution**: Execute scripts with multiple interpreter support
- **SSH Key Management**: Generate and manage SSH keys with GitHub integration
- **Extensible Architecture**: Add new package managers through TOML-based provider configurations

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

These scripts will download and install the latest version of RWR appropriate for your system. They will also set up the necessary paths and permissions.

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
   [`rwr config --create`](docs/cli/configuration.md). RWR asks for the settings
   and writes the file.
2. **Give the blueprints**: Give a git repository URL or a local path when RWR
   asks for it.
3. **Set up the system**: Run [`rwr all`](docs/cli/command-and-flags.md). RWR
   applies the blueprints.

For detailed setup instructions, see the [Quick Start Guide](docs/quick-start.md).

## Configuration

RWR uses a configuration file to manage settings like blueprint repositories, SSH keys, and GitHub API tokens. The configuration file supports both Git repositories and local filesystem paths for blueprints.

Basic configuration structure:

```yaml
repository:
  type: git
  url: "https://github.com/your-org/your-blueprints.git"
  # OR for local:
  # type: local
  # path: "/path/to/blueprints"
```

See the [Configuration documentation](docs/cli/configuration.md) for complete setup details.

## Profile System

RWR uses an additive profile system where:

- **Base items** (no profiles specified) are always applied
- **Profile items** are only applied when their profiles are active
- Multiple profiles can be activated simultaneously

```bash
# Apply base configuration only
rwr all

# Apply base + dev profile
rwr all --profile dev

# Apply base + multiple profiles
rwr all --profile dev --profile work
```

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

- **packages** — Install and remove packages with a package manager
- **repositories** — Manage the package repositories
- **files** — Copy, move, delete, and link files. The `directories` and
  `templates` keys are part of this type
- **services** — Manage the system services
- **configuration** — Set the desktop and system settings
- **git** — Clone and update git repositories
- **scripts** — Run scripts with a program that you select
- **users** — Manage the user accounts and the groups
- **ssh_keys** — Make SSH keys and send them to GitHub
- **fonts** — Install fonts
- **bootstrap** — Prepare the system before the other types run

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

### Blueprints

- [Blueprint Best Practices](docs/best-practices.md)
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

    This installs Go, GoReleaser, gotestsum, golangci-lint, and the other tools
    that the tasks use.

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

Update the dependencies:

```bash
mise run update     # go get -u ./... and go mod tidy
```

### CI/CD Pipeline

GitHub Actions runs the pipeline. There are three workflows.

`Go Build & Test` runs at each pull request:

- The `ci` job builds the code and runs the tests.
- The `security` job runs gosec and govulncheck.
- The `examples` job checks each file in `examples/`: the file parses in its
  format, the template variables exist, the fields match the blueprint structs,
  and the YAML, JSON and TOML copies give the same result. The job then runs
  `rwr validate` over each example tree.

`RWR - Master Prerelease` runs at each merge to `master`. It builds the branch
and publishes the files under the `nightly` tag. Read [Install](docs/install.md)
for more information.

`RWR - Release` runs at each version tag. It builds the binaries and the
packages, creates the GitHub release, and updates the Homebrew tap.

#### Environment Variables

The release workflow needs these secrets:

- `GITHUB_TOKEN` — creates the release and sends the files to it.
- `HOMEBREW_TAP_DEPLOY_KEY` — an SSH key that can write to the Homebrew tap
  repository.
- `NAUR_DISPATCH_TOKEN` — starts the update of the naur repository.

## Contributing

Contributions to RWR are welcome! If you'd like to contribute, please follow these steps:

1. Fork the repository on GitHub.
2. Create a new branch for your feature or bug fix.
3. Make your changes and commit them with descriptive commit messages.
4. Push your changes to your forked repository.
5. Submit a pull request to the main repository.

Please ensure that your code follows the project's coding style and includes appropriate tests.

## License

RWR is open-source software licensed under the [MIT License](LICENSE).

## Contact

If you have any questions, suggestions, or feedback, please open an issue on the [GitHub repository](https://github.com/fynxlabs/rwr/issues) or contact the maintainers directly.

Happy distrohopping with RWR!
