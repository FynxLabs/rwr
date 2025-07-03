# Rinse, Wash, Repeat (RWR)

![RWR Logo](img/rwr_128.gif)

Rinse, Wash, Repeat (RWR) is a powerful and flexible configuration management tool designed for those who like to hop around and reinstall frequently, regardless of whether it's Linux, macOS, or Windows. It aims to simplify the process of setting up and maintaining your system, making it easy to rebuild and reproduce configurations across multiple machines.

## Features

- **Blueprint-based Configuration**: Uses configuration files called blueprints to define and manage your system's configuration
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

RWR packages are available for various platforms and architectures through goreleaser. You can find the pre-built packages on the [releases page](https://github.com/fynxlabs/rwr/releases) of the RWR repository.

The following package types are available:

- Binary archives (`.tar.gz`, `.zip`)
- Debian packages (`.deb`)
- RPM packages (`.rpm`)
- Homebrew taps

### From Releases

To install RWR, follow these steps:

1. Download the latest release of RWR from the [releases page](https://github.com/fynxlabs/rwr/releases).
2. Extract the downloaded archive to a directory of your choice.
3. Add the directory to your system's `PATH` environment variable.

## Getting Started

1. **Initialize configuration**: Run [`rwr config init`](docs/cli/configuration.md) to create your configuration file
2. **Set up blueprints**: Provide a Git repository URL or local path for your blueprints during configuration
3. **Initialize system**: Run [`rwr all`](docs/cli/command-and-flags.md) to apply your blueprints

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
rwr all --profiles dev

# Apply base + multiple profiles
rwr all --profiles dev,work
```

For detailed information, see the [Profile System documentation](docs/profiles.md).

## Blueprint Types

RWR supports these blueprint types:

- **packages** - Package installation/removal via various package managers
- **repositories** - Package repository management
- **files** - File operations (copy, move, delete, symlink, templates)
- **directories** - Directory management with permissions
- **services** - System service management
- **configuration** - System configuration settings
- **git** - Git repository cloning and management
- **scripts** - Script execution with multiple interpreter support
- **users** - User account and group management
- **bootstrap** - Initial system setup tasks
- **ssh_keys** - SSH key generation and management

For detailed blueprint documentation, see the [Blueprint Types](docs/index.md#blueprints) section.

## Documentation

For detailed documentation on how to use RWR, please refer to the `docs/` directory. Here's an overview of the topics covered:

### Documentation Index

- [Documentation Index](docs/index.md)
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

2. Install required tools:

    ```bash
    mise install
    ```

This installs:

- Go (for building and testing)
- Dagger (for CI/CD pipeline)
- GoReleaser (for creating releases)
- gotestsum (for beautiful test output formatting and CI integration)

### Development Commands

RWR provides several commands for different development scenarios:

#### Local Development (No Dagger)

Fast commands that run directly on your machine:

```bash
# Build the binary
mise run build

# Run tests with beautiful formatting (uses gotestsum)
mise run test

# Run raw tests without formatting
mise run test:raw
```

#### Unit Testing Commands

For targeted testing of specific components, all with beautiful gotestsum formatting:

```bash
# Run all internal package tests (package-level summary)
mise run test:unit

# Run tests for specific packages (detailed test names)
mise run test:helpers     # Test helper functions
mise run test:processors  # Test blueprint processors
mise run test:system     # Test system utilities

# Run tests with coverage report
mise run test:coverage

# Watch mode - automatically run tests when files change
mise run test:watch
```

For raw test output without formatting, use the `:raw` variants:

```bash
# Raw test commands (no gotestsum formatting)
mise run test:unit:raw
mise run test:helpers:raw
mise run test:processors:raw
mise run test:system:raw
```

> [!NOTE]
> gotestsum provides superior test output formatting, watch mode for development, and saves test results to `/tmp/gotest.json` for CI integration and analysis.

#### Pipeline Testing (Using Dagger)

Test the CI pipeline locally using Dagger:

```bash
# Just run tests through Dagger
mise run dagger:test

# Test full release pipeline without publishing
# This will:
# 1. Run tests
# 2. Build binaries
# 3. Create archives
# 4. Skip actual publishing
mise run dagger:local

# Run full CI pipeline with publishing
# Requires:
# - GITHUB_TOKEN for creating releases
# - HOMEBREW_TAP_DEPLOY_KEY for updating Homebrew tap
mise run dagger:ci
```

#### Individual Dagger Functions

You can also run individual Dagger functions for specific tasks:

```bash
# Build binary through Dagger
mise run dagger:build

# Run linting through Dagger
mise run dagger:lint

# Test release process locally (without publishing)
mise run dagger:release

# Get version information
mise run dagger:version

# Clean Dagger generated files (useful for dependency issues)
mise run dagger:clean
```

### CI/CD Pipeline

RWR uses Dagger to manage its CI/CD pipeline. The pipeline:

1. On every push and PR:
   - Runs tests through Dagger
   - Reports test results

2. On version tags (v*):
   - Runs tests
   - Creates GitHub release with:
     - Binary archives (.tar.gz, .zip)
     - Debian packages (.deb)
     - RPM packages (.rpm)
   - Updates Homebrew tap

The pipeline can be tested locally using the commands above, making it easy to verify changes before pushing.

#### Environment Variables

For publishing releases:

- `GITHUB_TOKEN`: GitHub token with permissions to:
  - Create releases
  - Upload release assets
  - Update repository contents
- `HOMEBREW_TAP_DEPLOY_KEY`: SSH key with access to update the Homebrew tap repository

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
