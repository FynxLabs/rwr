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
* Verifying cross-references between blueprints and providers
* Ensuring system compatibility

## Command Flags

| Flag | Description |
|------|-------------|
| `--blueprints` | Validate only blueprint files |
| `--providers` | Validate only provider configurations |
| `--all` | Validate everything (default) |
| `--path string` | Path to validate (default current directory) |
| `--verbose` | Show detailed validation information |

## Validation Process

### Blueprint Validation

Blueprint validation checks your blueprint files for structural and content issues.

The blueprint validation process includes:

* **Structure Validation**: Ensures all required fields are present and field types match specifications
* **Content Validation**: Checks that file paths exist or can be created, permissions are valid, and commands are executable
* **Cross-Reference Validation**: Verifies that referenced package managers, files, and services exist

### Provider Validation

Provider validation checks your provider configuration files for structural and compatibility issues.

The provider validation process includes:

* **Structure Validation**: Ensures all required sections exist and field types match specifications
* **Command Validation**: Checks command syntax and template variable usage
* **System Compatibility**: Verifies that the provider supports the current OS/distribution and required binaries exist

### Cross-Reference Validation

Cross-reference validation ensures that all references between blueprints and providers are valid.

The cross-reference validation process includes:

* **Package Manager References**: Verifies that each referenced package manager has a valid provider
* **Dependency Validation**: Checks for circular dependencies and ensures all dependencies exist
* **Path Validation**: Verifies that file paths exist or can be created and have appropriate permissions

## Error Reporting

The validate command provides detailed error reports to help you identify and fix issues in your configurations.

Error reports include:

* **Error Messages**: Detailed descriptions of validation errors
* **Warning Messages**: Potential issues that might not cause failures but could be improved
* **Suggestions**: Recommended fixes for identified issues
* **File and Line References**: Exact locations of issues in your configuration files

## Examples

### Validating All Configurations

To validate all blueprints and provider configurations in the current directory:

```bash
rwr validate
```

### Validating Only Blueprints

To validate only blueprint files:

```bash
rwr validate --blueprints
```

### Validating Only Providers

To validate only provider configurations:

```bash
rwr validate --providers
```

### Validating a Specific Path

To validate configurations in a specific directory:

```bash
rwr validate --path /path/to/configs
```

### Verbose Output

To get detailed validation information:

```bash
rwr validate --verbose
```

## Common Validation Errors

### Blueprint Errors

| Error | Description |
|-------|-------------|
| Missing required field | A required field is missing in a blueprint |
| Invalid field type | A field has an incorrect type |
| Invalid package manager reference | A referenced package manager does not exist |
| Invalid file path | A file path does not exist or cannot be created |
| Invalid permission | A permission value is invalid |

### Provider Errors

| Error | Description |
|-------|-------------|
| Missing required section | A required section is missing in a provider configuration |
| Invalid command syntax | A command has invalid syntax |
| Invalid template variable | A template variable is used incorrectly |
| Unsupported distribution | The provider does not support the current distribution |
| Missing binary | A required binary is not available |

## Best Practices

* Run validation before attempting to run your configurations
* Use the `--verbose` flag to get detailed information about validation issues
* Fix all errors and warnings before proceeding with deployment
* Validate both blueprints and providers to ensure complete compatibility
* Use the validation output to improve your configuration files

## See Also

* [Commands and Flags](command-and-flags.md)
* [Blueprints Overview](../blueprints-general.md)
* [Providers](../providers.md)
* [Best Practices](../best-practices.md)
