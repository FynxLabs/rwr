# Quick Start Guide

This guide will help you get started with Rinse, Wash, Repeat (RWR) quickly. You'll learn how to install RWR, set up a basic configuration, and run your first blueprint.

## Prerequisites

Before you begin, ensure that you have the following:

- A supported operating system (Linux, macOS, or Windows)
- A compatible package manager (e.g., apt, brew, chocolatey)
- Git installed on your system

## Installation

To install RWR, follow these steps:

1. Download the latest release of RWR from the [releases page](https://github.com/fynxlabs/rwr/releases).
2. Extract the downloaded archive to a directory of your choice.
3. Add the directory to your system's `PATH` environment variable.

## Configuration

To set up a basic configuration for RWR, follow these steps:

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

2. Run the `rwr all` command to execute all blueprints:

    ```bash
    rwr all
    ```

    This will install only the base packages (git and curl).

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

RWR will now process the `packages.yaml` blueprint and install the appropriate packages based on your selected profiles.

## Next Steps

Congratulations! You have successfully installed RWR, set up a basic configuration, and run your first blueprint.

Next, you can:

- Learn about the [Profile System](profiles.md) to organize your configurations for different contexts and environments.
- Explore the [Blueprints Overview](blueprints-general.md) to learn more about the different blueprint types and their capabilities.
- Customize your configuration by adding more blueprints and adjusting the `init.yaml` file.
- Learn how to use [Variables](variables.md) to make your blueprints more dynamic and reusable.
- Discover [Best Practices](best-practices.md) for organizing and managing your RWR configurations.
- Review [Profile Best Practices](profile-best-practices.md) for practical organizational tips.

If you encounter any issues or have questions, please refer to the troubleshooting section or reach out to the RWR community for support.
