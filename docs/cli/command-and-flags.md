# Commands and Flags

The Rinse, Wash, Repeat (RWR) CLI provides a set of commands and flags to manage your system's configuration. This page describes the available commands and their associated flags.

## Global Flags

The following flags are available for all commands:

| Flag | Description |
|------|-------------|
| `--debug`, `-d` | Enable debug mode for verbose output |
| `--log-level` | Set the log level (debug, info, warn, error) |
| `--init-file`, `-i` | Specify the path to the `init.yaml` file |
| `--gh-api-key` | Specify the GitHub API key for accessing private repositories |
| `--ssh-key` | Specify the SSH key (base64 encoded) for accessing private repositories |
| `--skip-version-check` | Skip checking for the latest version of RWR |
| `--show-secrets` | Print credential values in logs instead of redacting them. Off by default; use it only to confirm RWR is reading the token or key you expect |
| `--dry-run` | Simulate operations without making changes (no-op mode) |
| `--no-op` | Alias for `--dry-run` |
| `--interactive`, `-I` | Enable interactive mode (default: true). Use `--interactive=false` to disable |
| `--gh-key` | Another name for `--gh-api-key` |
| `--gh-auth` | Get a GitHub token with the OAuth device flow |
| `--profile`, `-p` | Make a profile active. Use the flag one time for each profile |
| `--force-bootstrap` | Run the bootstrap process again |

## Commands

The following commands are available in the RWR CLI:

### `rwr config`

Manage RWR configuration settings.

| Flag | Description |
|------|-------------|
| `--create`, `-c` | Create the configuration file |

### `rwr all`

Run all blueprints and set up the system.

`--force-bootstrap` is a global flag. It also applies to this command.

### `rwr validate`

Check the RWR blueprints and the provider configurations.

Give the path as an argument, not as a flag:

```bash
rwr validate path/to/blueprints
```

| Flag | Description |
|------|-------------|
| `--blueprints` | Check the path as blueprint files |
| `--providers` | Check the path as provider configurations |
| `--verbose` | Show more information about each check |

### `rwr profiles`

Show the profiles that the blueprints use.

This command has no flags.

### `rwr run`

Run individual processors.

#### `rwr run packages`

Run the package processor.

#### `rwr run repository`

Run the repository processor.

#### `rwr run services`

Run the services processor.

#### `rwr run files`

Run the files processor. This covers `files:`, `directories:` and `templates:`,
which all live in a files blueprint.

#### `rwr run configuration`

Run the configuration processor.

#### `rwr run git`

Run the Git repository processor.

#### `rwr run scripts`

Run the scripts processor.

#### `rwr run ssh_keys`

Run the SSH key processor. Use `--gh-auth` with this command to get a GitHub
token first.

#### `rwr run users`

Run the users and groups processor.

#### `rwr run fonts`

Run the fonts processor.

> [!NOTE]
> There is no `rwr run directories` command. The `directories` key is part of a
> files blueprint. Use `rwr run files` to process it.

## Examples

Here are a few examples of using the RWR CLI with different commands and flags:

```bash
# Initialize the system with debug mode enabled
rwr all --debug

# Run the package processor with a specific init file
rwr run packages --init-file path/to/init.yaml

# Create the configuration file
rwr config --create

# Run all blueprints with bootstrap forced
rwr all --force-bootstrap
```

For more detailed information on each command and its usage, please refer to the specific blueprint type documentation or the [Configuration File](configuration.md) page.
