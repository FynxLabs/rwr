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

## Commands

The following commands are available in the RWR CLI:

### `rwr config`

Manage RWR configuration settings.

| Flag | Description |
|------|-------------|
| `--create`, `-c` | Create the configuration file |

### `rwr all`

Initialize the system by running all blueprints.

| Flag | Description |
|------|-------------|
| `--force-bootstrap` | Force the bootstrap process to run again |

### `rwr validate`

Validate the RWR blueprints.

### `rwr run`

Run individual processors.

#### `rwr run package`

Run the package processor.

#### `rwr run repository`

Run the repository processor.

#### `rwr run services`

Run the services processor.

#### `rwr run files`

Run the files processor.

#### `rwr run directories`

Run the directories processor.

#### `rwr run configuration`

Run the configuration processor.

#### `rwr run git`

Run the Git repository processor.

#### `rwr run scripts`

Run the scripts processor.

#### `rwr run ssh_keys`

Run the scripts processor.

#### `rwr run users`

Run the users and groups processor.

## Examples

Here are a few examples of using the RWR CLI with different commands and flags:

```bash
# Initialize the system with debug mode enabled
rwr all --debug

# Run the package processor with a specific init file
rwr run package --init-file path/to/init.yaml

# Create the configuration file
rwr config --create

# Run all blueprints with bootstrap forced
rwr all --force-bootstrap
```

For more detailed information on each command and its usage, please refer to the specific blueprint type documentation or the [Configuration File](configuration.md) page.
