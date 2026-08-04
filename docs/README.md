# RWR Documentation

Map of the Rinse, Wash, Repeat (RWR) docs. Start here if you are new.

## Start here

- [Documentation Home](index.md) — what RWR is and an overview of every topic.
- [Install RWR](install.md) — install with a script, a package manager, or a binary.
- [Quick Start Guide](quick-start.md) — install, make a basic configuration, run your first blueprint.

## CLI

- [CLI docs](cli/README.md) — commands, flags, the configuration file, the profiles CLI, `rwr validate`, and `rwr convert`.

## Blueprints

- [What are Blueprints?](blueprints-general.md) — the blueprint model: types, formats, and structure.
- [Blueprint type docs](blueprints/README.md) — one page per blueprint type (packages, files, services, ...).
- [Init File](init-file.md) — the entry point that names your blueprint tree and its settings.
- [Bootstrap Process](bootstrap.md) — the setup RWR runs before the main blueprints.
- [Blueprint schema versioning](schema-versioning.md) — how blueprints declare a schema version per type.

## Profiles

- [Profile System](profiles.md) — group packages and configurations by context and select what applies.
- [Profile Best Practices](profile-best-practices.md) — practical tips and examples for organizing profiles.

## Variables

- [Variables and Templating](variables.md) — variables and templates so one blueprint serves many machines.

## Providers

- [Package Manager Providers](providers.md) — declarative definitions of each package manager, no Go code needed: authored in CUE, overridable with TOML or JSON files.

## Credentials and security

- [Credentials in blueprints](credentials.md) — the credentials RWR holds (GitHub token, SSH key, and any you declare), where they are sourced from, and why blueprints cannot read them by default.

## Run records

- [Run records](state.md) — the journal each run writes, `rwr status` for drift, and `rwr uninstall` to reverse a run.

## General

- [Best Practices](best-practices.md) — blueprint organization, configuration management, and maintenance.
