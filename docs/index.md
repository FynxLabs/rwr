# Rinse, Wash, Repeat (RWR) Documentation

Welcome to the Rinse, Wash, Repeat (RWR) documentation! This guide aims to provide comprehensive information about using and extending the RWR configuration management tool.

## Introduction

RWR is a powerful and flexible configuration management tool designed for those who like to hop around and reinstall frequently, regardless of whether it's Linux, macOS, or Windows. It simplifies the process of setting up and maintaining your system by using blueprint-based configurations with an advanced profile system for selective installation.

## Quick Start Guide

- [Quick Start Guide](quick-start.md): Get up and running with RWR quickly by following this concise guide.

## Profile System

RWR's profile system allows you to organize and selectively install packages and configurations based on different contexts, environments, or use cases.

- [Profile System Overview](profiles.md): Complete guide to understanding and using profiles
- [Profile CLI Commands](cli/profiles.md): Command-line reference for profile usage
- [Profile Best Practices](profile-best-practices.md): Practical organizational tips and examples

## How the CLI Works

- [Commands and Flags](cli/command-and-flags.md): Learn about the available commands in the RWR CLI and their respective flags.
- [Configuration File](cli/configuration.md): Understand how to configure RWR through the configuration file.
- [Profile Commands](cli/profiles.md): Profile-specific CLI commands and flags.

## The Init File

The [Init File](init-file.md) is the main entry point for your blueprints and defines the order of execution. This section will cover its structure and functionality.

## The Bootstrap Process

The [Bootstrap Process](bootstrap.md) is responsible for setting up the initial system configuration. Learn how it works and how to define the bootstrap file.

## Blueprints Overview

Get a [general overview of Blueprints](blueprints-general.md) and how they are used to manage your system's configuration.

## Blueprint Types

RWR supports various blueprint types for managing different aspects of your system. Each blueprint type has its own page with detailed information:

- [Packages Blueprint](blueprints/packages.md)
- [Repositories Blueprint](blueprints/repositories.md)
- [Configuration Blueprint](blueprints/configuration.md)
- [Files Blueprint](blueprints/files.md)
- [Directories Blueprint](blueprints/directories.md)
- [Services Blueprint](blueprints/services.md)
- [Users and Groups Blueprint](blueprints/users-and-groups.md)
- [Git Blueprint](blueprints/git.md)
- [Scripts Blueprint](blueprints/scripts.md)
- [SSH Keys Blueprint](blueprints/ssh-keys.md)
- [Fonts Blueprint](blueprints/fonts.md)

## Variables and Templating

- [Variables and Templating](variables.md): Learn how to use variables and templating in blueprints to make them more dynamic and reusable.

## Best Practices

- [Best Practices](best-practices.md): Discover best practices and recommendations for organizing blueprints and managing configurations.

## Extending RWR

- Coming Soon: Adding a New Processor

## Troubleshooting

- Coming Soon: Troubleshooting section for common issues and solutions.

## Additional Resources

- Coming Soon: Frequently Asked Questions (FAQ)
- Coming Soon: Known Issues
- Coming Soon: Glossary
