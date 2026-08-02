# Validate Command

The `validate` command in RWR checks your blueprints and provider configurations before you run them. This check finds errors before you deploy.

## Overview

The `validate` command checks your blueprints and provider configurations. It finds errors before you run them.

```bash
rwr validate [flags]
```

The validation process does these checks:

* It checks blueprint structure and content
* It checks provider configurations
* It resolves blueprint imports and detects circular imports
* It checks that a referenced package manager has a provider

> [!NOTE]
> `validate` reads files. It never installs anything and never runs a blueprint's
> commands, so it cannot tell you that a package exists in a repository or that a
> script succeeds.
>
> `--profile` is accepted, because it is a global flag, but `validate` ignores
> it: every blueprint file in the tree is checked regardless of profiles.

## Command Flags

| Flag | Description |
|------|-------------|
| `--blueprints` | Force validation as blueprint files |
| `--providers` | Force validation as provider configurations |
| `--verbose` | Log a "validation completed" line when the walk finishes |

Give the path as an argument, not as a flag. The default is the current
directory:

```bash
rwr validate path/to/blueprints
```

With neither `--blueprints` nor `--providers`, RWR chooses one from the path.
RWR validates a directory named `providers` or a `.toml` file as providers. RWR
validates all other paths as blueprints. Each run validates one of the two,
never both.

Blueprint validation needs an init file (`init.json`, `init.yaml`, `init.yml`
or `init.toml`) at or above the checked files. RWR cannot validate a tree with
no init file.

> [!NOTE]
> `--verbose` does not change what is checked or add per-file output. Use
> `--debug` for a detailed trace.

## Validation Process

### Blueprint Validation

Blueprint validation checks your blueprint files for structural and content errors.

The blueprint validation process includes:

* **Structure Validation**: Each file decodes strictly into its blueprint type. An
  unknown key is an error, not something silently ignored
* **Required Fields**: Required fields such as a repository `url`, an SSH key
  `name`, `type` and `path`, or a file's `content` or `source`, must be present
* **Enumerations**: `action` values are checked against the actions the processor
  accepts, and file modes against what RWR will accept
* **Imports**: Each `import` path must exist and must parse as the expected
  blueprint type. A circular import is an error
* **Package Managers**: A `package_manager` named by a blueprint must have a
  provider available on this system

### Provider Validation

Provider validation checks your provider configuration files for structural and compatibility errors.

The provider validation process includes:

* **File Type**: A provider configuration must be a `.toml` file
* **Required Fields**: `provider.name`, the detection `binary` and
  `distributions`, and the `install`, `update` and `remove` commands must all be
  present
* **Steps**: Each install step must name an action
* **Paths**: A declared repository sources path must exist

## Error Reporting

RWR logs each issue when it finds it, with the source file and a suggested
fix:

```text
ERRO ERROR: No URL specified for repository 'core' [/path/to/repositories/main.yaml] - Suggestion: Add URL field to repository
WARN WARNING: Location does not exist: blueprints [/path/to/init.yaml] - Suggestion: Create the directory or update the location
```

The command ends with a summary, and exits non-zero if there was at least one
error:

```text
validation failed with 2 errors and 1 warnings
```

With no errors it prints `Validation completed with N warnings`, or
`Validation completed successfully`.

> [!NOTE]
> Issues carry the file, not a line number. Where the underlying decoder reports
> a line — a strict-decoding failure, for example — it appears inside the message
> text.

## Examples

### Validating the Current Directory

```bash
rwr validate
```

### Validating a Specific Path

```bash
rwr validate /path/to/blueprints
rwr validate providers/paru.toml
```

### Forcing a Mode

```bash
rwr validate path/to/dir --blueprints
rwr validate path/to/file --providers
```

## Common Validation Errors

### Blueprint Errors

| Message | Meaning |
|---------|---------|
| `field <name> not found in type types.<T>` | The blueprint uses a key that type does not have. Blueprints decode strictly, so this is an error rather than a silently ignored key |
| `No URL specified for repository '<name>'` | A repository entry has no `url` |
| `No content or source specified for file '<name>'` | A file entry gives neither |
| `No type specified for SSH key '<name>'` | An SSH key entry has no `type` |
| `Circular import detected: '<path>'` | An import chain returns to a file it already read |
| `Import file not found '<path>'` | An `import` path does not resolve |
| `Location does not exist: <path>` | The init file's `blueprints.location` is not there (a warning) |
| `Failed to find init file` | There is no `init.*` at or above the path being validated |

### Provider Errors

| Message | Meaning |
|---------|---------|
| `Not a TOML file` | Provider configurations must be `.toml` |
| `Missing required field 'provider.name'` | The `[provider]` section has no `name` |
| `Missing binary in detection section` | The provider does not say which binary to look for |
| `No distributions specified in detection section` | The provider does not say which distributions it supports |
| `Missing install command` / `Missing update command` / `Missing remove command` | A required command is absent from `[commands]` |
| `No providers available for the current system` | Nothing was detected on this machine |

## Best Practices

* Validate your configurations before you run them
* Fix all errors and warnings before you deploy
* Validate both blueprints and providers, in two runs, to cover both
* Use `--debug` when a message is not enough to locate the error

## See Also

* [Commands and Flags](command-and-flags.md)
* [Blueprints Overview](../blueprints-general.md)
* [Providers](../providers.md)
* [Best Practices](../best-practices.md)
