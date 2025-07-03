# Configuration File

The Rinse, Wash, Repeat (RWR) configuration file (`config.yaml`) is used to store settings and preferences for the RWR tool. This page describes the structure and options available in the configuration file.

## File Location

The `config.yaml` file is located in the RWR configuration directory. By default, the configuration directory is located at:

- Linux and macOS: `$HOME/.config/rwr`
- Windows: `%USERPROFILE%\.config\rwr`

You can also specify a custom location for the configuration file using the `--config` flag when running RWR commands.

## File Format

The configuration file uses the YAML format. It consists of key-value pairs and nested sections to organize the settings.

## Configuration Options

The following options are available in the `config.yaml` file:

### `rwr` Section

The `rwr` section contains general settings for the RWR tool.

| Option | Description |
|--------|-------------|
| `configdir` | Specifies the directory where the configuration file is located |
| `skipVersionCheck` | Skips checking for the latest version of RWR when set to `true` |

### `repository` Section

The `repository` section contains settings related to Git repositories.

| Option | Description |
|--------|-------------|
| `gh_api_token` | Specifies the GitHub API token for accessing private repositories |
| `ssh_private_key` | Specifies the SSH private key (base64 encoded) for accessing private repositories |
| `init-file` | Specifies the location of the init file (local or URL) |

### `packageManager` Section

The `packageManager` section allows you to set the default package manager for each supported operating system.

| Option | Description |
|--------|-------------|
| `linux.default` | Specifies the default package manager for Linux |
| `macos.default` | Specifies the default package manager for macOS |
| `windows.default` | Specifies the default package manager for Windows |

### `log` Section

The `log` section contains settings related to logging.

| Option | Description |
|--------|-------------|
| `level` | Specifies the log level (debug, info, warn, error) |

## Example Configuration File

Here's an example `config.yaml` file:

```yaml
rwr:
  configdir: /path/to/custom/config
  skipVersionCheck: false

repository:
  gh_api_token: your_github_api_token
  ssh_private_key: your_ssh_private_key_base64
  init-file: https://example.com/init.yaml

packageManager:
  linux:
    default: apt
  macos:
    default: brew
  windows:
    default: chocolatey

log:
  level: info
```

## Modifying the Configuration File

You can modify the `config.yaml` file directly using a text editor. Alternatively, you can use the `rwr config` command to interactively create or update the configuration file.

```bash
rwr config --create
```

This command will prompt you for the necessary settings and generate the `config.yaml` file based on your input.

## Precedence

The settings in the `config.yaml` file have precedence over the default values used by RWR. However, command-line flags, when provided, will override the corresponding settings in the configuration file.

## Environment Variables

RWR also supports setting configuration options through environment variables. Environment variables take precedence over the `config.yaml` file but are overridden by command-line flags. To set an option using an environment variable, use the prefix `RWR_` followed by the uppercase version of the option name, with dots replaced by underscores.

For example:

- `RWR_LOG_LEVEL=debug` sets the log level to debug
- `RWR_REPOSITORY_GH_API_TOKEN=your_token` sets the GitHub API token

## Notes

- The `ssh_private_key` in the `repository` section is used as the default SSH key for RWR operations, including private git clones. This key is set when an SSH key is generated with the `set_as_rwr_ssh_key: true` option in the SSH Keys blueprint.
- When using URL sources for files or init files, RWR will download the file from the specified URL before processing it.

For more information on using the configuration file and its options, please refer to the [Commands and Flags](command-and-flags.md) and [Best Practices](../best-practices.md) guides.
