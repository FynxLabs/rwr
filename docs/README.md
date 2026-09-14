# RWR documentation

Start with the path that matches what you are trying to do. Reference pages are
there when you need exact fields or flags; you do not need to read the whole
manual before your first run.

## Set up RWR for the first time

1. [Install RWR](install.md)
2. [Run a small local blueprint](quick-start.md)
3. [Learn how a blueprint tree is organized](blueprints-general.md)
4. [Connect RWR to a saved local or Git-hosted setup](cli/configuration.md)

Already have a configured machine? Use [`rwr capture`](cli/capture.md) to create
a starting tree from it.

## Build a blueprint tree

- [Init file](init-file.md) — choose the tree, format, run order, profiles,
  package managers, and credential policy.
- [Blueprint types](blueprints/README.md) — find the schema for packages, files,
  services, users, configuration tools, and the other processors.
- [Common fields](blueprints/common-fields.md) — profiles, imports, interaction,
  and strict field checking.
- [Variables and templates](variables.md) — reuse machine and user values.
- [Examples](../examples/README.md) — working trees in every supported format.
- [Best practices](best-practices.md) — patterns for keeping a growing repository
  understandable.

## Run and inspect a setup

- [Commands and flags](cli/command-and-flags.md) — the complete CLI reference.
- [Profiles](profiles.md) — select subsets of one shared setup.
- [Validation](cli/validate.md) — catch structural mistakes before applying.
- [Diff](cli/diff.md) — turn current-machine differences into blueprint material.
- [Run records](state.md) — inspect drift and reverse supported changes.
- [Bootstrap](bootstrap.md) — prepare prerequisites before ordinary processors.

## Configure specialized features

- [Credentials and Bitwarden](credentials.md) — provision first, then set up
  providers, keyring entries, and GPG tasks explicitly.
- [Omarchy configuration tool](blueprints/omarchy.md) — use the configuration
  blueprint to manage plugins, shell settings, themes, defaults, hooks, and
  external screensaver routing after login.
- [Package manager providers](providers.md) — understand or override the
  declarative package-manager definitions embedded in RWR.
- [Schema versioning](schema-versioning.md) — pin and migrate blueprint schemas.
- [Convert](cli/convert.md) — move a tree between YAML, JSON, TOML, and CUE.

For a broad overview, return to the [project README](../README.md). For command
discovery at the terminal, run `rwr help` or `rwr help COMMAND`.
