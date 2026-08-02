# Quick Start Guide

This guide tells you how to install Rinse, Wash, Repeat (RWR), make a basic configuration, and run your first blueprint.

## Prerequisites

Before you start, make sure that you have these items:

- A supported operating system (Linux, macOS, or Windows)
- A compatible package manager (for example apt, brew, or chocolatey)
- Git installed on your system

## Installation

To install RWR, follow these steps:

1. Download the latest release of RWR from the [releases page](https://github.com/fynxlabs/rwr/releases).
2. Extract the downloaded archive to a directory of your choice.
3. Add the directory to your system's `PATH` environment variable.

## Configuration

To make a basic configuration for RWR, follow these steps:

1. Create a new directory for your RWR configuration:

    ```bash
    mkdir my-rwr-config
    cd my-rwr-config
    ```

2. Create an `init.yaml` file with the following content:

    ```yaml
    blueprints:
      format: yaml
      location: blueprints
    ```

3. Create a `blueprints` directory:

    ```bash
    mkdir blueprints
    ```

4. Inside the `blueprints` directory, create a `packages.yaml` file with the following content:

```yaml
packages:
  # Base packages - always installed (using names for multiple packages)
  - names: [git, curl, wget, htop]
    action: install

  # Development profile packages (mix of single and multiple)
  - names: [docker, nodejs, npm, python3]
    action: install
    profiles: [development]
  - name: code
    action: install
    profiles: [development, web]

  # Work profile packages
  - names: [slack, zoom, teams]
    action: install
    profiles: [work]
```

## Running Your First Blueprint

To run your first blueprint, follow these steps:

1. Open a terminal and navigate to your RWR configuration directory:

    ```bash
    cd my-rwr-config
    ```

2. Run the `rwr all` command to apply all blueprints:

    ```bash
    rwr all
    ```

    RWR installs only the base packages: git, curl, wget, and htop.

3. To install packages for specific profiles, use the `--profile` flag:

    ```bash
    # Install base packages + development profile
    rwr all --profile development

    # Install base packages + work profile
    rwr all --profile work

    # Install multiple profiles
    rwr all --profile development --profile work
    ```

4. To see what profiles are available in your configuration:

    ```bash
    rwr profiles
    ```

RWR processes the `packages.yaml` blueprint. It installs the packages for the profiles that you selected.

## Next Steps

You installed RWR, made a basic configuration, and ran your first blueprint.

Next, you can:

- Read the [Profile System](profiles.md) page to organize your configurations for different contexts and environments.
- Read the [Blueprints Overview](blueprints-general.md) for the different blueprint types and their capabilities.
- Read the [Variables](variables.md) page to make your blueprints more dynamic and reusable.
- Read the [Best Practices](best-practices.md) page for blueprint organization and configuration management.
- Read the [Profile Best Practices](profile-best-practices.md) page for practical tips on profile organization.
- Add more blueprints and adjust the `init.yaml` file.

If you have a problem, read the troubleshooting section. You can also contact the RWR community.
