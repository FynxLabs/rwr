# Validate Command

The `validate` command in RWR helps you verify your blueprints and provider configurations before running them. This ensures that your configurations are correct and will work as expected when deployed.

## Overview

The validate command performs comprehensive checks on your RWR blueprints and provider configurations to identify potential issues before you attempt to run them. This can save time and prevent errors during deployment.

```bash
rwr validate [flags]
```

The validation process includes:

* Checking blueprint structure and content
* Validating provider configurations
* Resolving blueprint imports and detecting circular ones
* Checking that a referenced package manager has a provider

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

With neither `--blueprints` nor `--providers`, RWR chooses one from the path: a
directory named `providers` or a `.toml` file is validated as providers, and
anything else as blueprints. Each run validates one of the two, never both.

Validating as blueprints needs an init file (`init.json`, `init.yaml`,
`init.yml` or `init.toml`) at or above the files being checked; a tree with no
init file cannot be validated.

> [!NOTE]
> `--verbose` does not change what is checked or add per-file output. Use
> `--debug` for a detailed trace.

## Validation Process

### Blueprint Validation

Blueprint validation checks your blueprint files for structural and content issues.

The blueprint validation process includes:

* **Structure Validation**: Each file decodes strictly into its blueprint type. An
  unknown key is an error, not something silently ignored
* **Required Fields**: Required fields such as a repository `url`, an SSH key
  `name`, `type` and `path`, or a file's `content` or `source`, must be present
* **Enumerations**: `action` values are checked against the actions the processor
  accepts, and file modes against what RWR will accept
* **Imports**: Each `import` path must exist and must parse as the expected
  blueprint type; a circular import is an error
* **Package Managers**: A `package_manager` named by a blueprint must have a
  provider available on this system

### Provider Validation

Provider validation checks your provider configuration files for structural and compatibility issues.

The provider validation process includes:

* **File Type**: A provider configuration must be a `.toml` or `.json` file
  (an error otherwise)
* **Required Fields**: A missing `provider.name` is an **error**. A missing
  detection `binary` or `distributions` list, or a missing `install`, `update`
  or `remove` command, is a **warning**
* **Steps**: Each install, remove and repository step must name an action the
  processor implements, and carry the fields that action needs (`exec` for
  `command`, `source`/`dest` for `download` and `copy`, and so on) — errors.
  Empty `content` on a `write` or `append` step is a warning
* **Paths**: A declared repository sources path that does not exist on this
  system is a warning

## Error Reporting

Each issue is logged as it is found, with the file it came from and a suggested
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

| Message | Severity | Meaning |
|---------|----------|---------|
| `Not a provider file` | Error | Provider configurations must be `.toml` or `.json` |
| `Missing required field 'provider.name'` | Error | The `[provider]` section has no `name` |
| `Missing binary in detection section` | Warning | The provider does not say which binary to look for |
| `No distributions specified in detection section` | Warning | The provider does not say which distributions it supports |
| `Missing install command` / `Missing update command` / `Missing remove command` | Warning | A command is absent from `[commands]` |
| `Unsupported action "…" in … step` | Error | A step names an action the processor does not implement |
| `Missing exec/source/dest/path/… in … step` | Error | A step lacks a field its action needs |
| `No providers available for the current system` | Warning | Nothing was detected on this machine |

Only errors make the exit code non-zero; a run that produces warnings alone
still exits `0`.

## Best Practices

* Run validation before attempting to run your configurations
* Fix all errors and warnings before proceeding with deployment
* Validate both blueprints and providers, in two runs, to cover both
* Use `--debug` when a message is not enough to locate the problem

## See Also

* [Commands and Flags](command-and-flags.md)
* [Blueprints Overview](../blueprints-general.md)
* [Providers](../providers.md)
* [Best Practices](../best-practices.md)
