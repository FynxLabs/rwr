# Rinse, Wash, Repeat (RWR) Documentation

Welcome to the Rinse, Wash, Repeat (RWR) documentation. This guide tells you how to use and how to extend RWR.

## Introduction

RWR is a configuration management tool for users who reinstall their systems frequently.

- It works on Linux, macOS, and Windows.
- Blueprints configure the system.
- Profiles select what RWR installs.

## Install and Quick Start

- [Install](install.md): The supported platforms, the install scripts, and the
  builds of the master branch.
- [Quick Start Guide](quick-start.md): This guide tells you how to install RWR and run your first blueprint.

## Profile System

With the profile system, you can install different packages for different contexts, environments, or use cases.

- [Profile System Overview](profiles.md): How profiles work and how to use them
- [Profile CLI Commands](cli/profiles.md): The command-line reference for profiles
- [Profile Best Practices](profile-best-practices.md): Practical tips and examples for profile organization

## How the CLI Works

- [Commands and Flags](cli/command-and-flags.md): The available commands in the RWR CLI and their flags.
- [Configuration File](cli/configuration.md): How to configure RWR through the configuration file.
- [Profile Commands](cli/profiles.md): Profile-specific CLI commands and flags.

## The Init File

The [init file](init-file.md) is the main entry point for your blueprints and defines the order of execution. This section describes its structure and function.

## The Bootstrap Process

The [bootstrap process](bootstrap.md) sets the initial system configuration. The page tells you how it works and how to define the bootstrap file.

## Blueprints Overview

A [general overview of Blueprints](blueprints-general.md), and how they manage the configuration of your system.

## Blueprint Types

RWR supports many blueprint types. Each blueprint type manages one part of your system and has its own page with detailed information:

- [Packages Blueprint](blueprints/packages.md)
- [Repositories Blueprint](blueprints/repositories.md)
- [Configuration Blueprint](blueprints/configuration.md)
- [Files Blueprint](blueprints/files.md)
- [Directories Blueprint](blueprints/directories.md) (a key in a files blueprint,
  not a separate type)
- [Services Blueprint](blueprints/services.md)
- [Users and Groups Blueprint](blueprints/users-and-groups.md)
- [Git Blueprint](blueprints/git.md)
- [Scripts Blueprint](blueprints/scripts.md)
- [SSH Keys Blueprint](blueprints/ssh-keys.md)
- [Fonts Blueprint](blueprints/fonts.md)

## Variables and Templating

- [Variables and Templating](variables.md): How to use variables and templates in blueprints.

## Credentials and Schema Versions

- [Credentials](credentials.md): How RWR holds your GitHub token and SSH key, and
  how to give a blueprint access to one.
- [Schema versioning](schema-versioning.md): How a blueprint gives the schema
  version that it uses, and how one blueprint type moves to a new version.

## Best Practices

- [Best Practices](best-practices.md): The best practices for blueprint organization and configuration management.

## Extending RWR

- Coming Soon: Adding a New Processor

## Troubleshooting

- Coming Soon: Troubleshooting section for common issues and solutions.

## Additional Resources

- Coming Soon: Frequently Asked Questions (FAQ)
- Coming Soon: Known Issues
- Coming Soon: Glossary
