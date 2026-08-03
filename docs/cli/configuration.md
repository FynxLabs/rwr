# Configuration File

The Rinse, Wash, Repeat (RWR) configuration file (`config.yaml`) is used to store settings and preferences for the RWR tool. This page describes the structure and options available in the configuration file.

## File Location

The `config.yaml` file is located in the RWR configuration directory. By default, the configuration directory is located at:

- Linux and macOS: `$HOME/.config/rwr`
- Windows: `%USERPROFILE%\.config\rwr`

`--config` is a global flag that changes where RWR looks. It accepts either:

- **a file** — RWR reads that file as the configuration, whatever it is named.
  The directory holding it becomes the configuration directory, so `run_once`
  is created next to it.
- **a directory** — RWR reads `config.yaml` from that directory and uses the
  directory as the configuration directory.

```bash
rwr all --config ~/rwr-work/config.yaml
rwr all --config ~/rwr-work
```

A leading `~` is expanded. RWR creates the configuration directory and its
`run_once` subdirectory if they do not exist. A missing configuration file is not
an error: RWR logs it at debug level and continues with the defaults.

## File Format

The configuration file uses the YAML format. It consists of key-value pairs and nested sections to organize the settings.

## Configuration Options

The following options are available in the `config.yaml` file:

### `rwr` Section

The `rwr` section contains general settings for the RWR tool.

| Option | Description |
|--------|-------------|
| `configdir` | The config directory: where `config.yaml` is looked up, where `rwr config --create` writes it, and where the `bootstrap` marker lives. The default is `$HOME/.config/rwr`. Because it decides where the config file is, it only takes effect from the environment (`RWR_RWR_CONFIGDIR`) — a config file cannot name its own directory. `--config` overrides it |
| `skipVersionCheck` | Set to `true` to turn off the startup check for a newer release. The `--skip-version-check` flag does the same for a single run |

### `repository` Section

The `repository` section contains settings related to Git repositories.

| Option | Description |
|--------|-------------|
| `gh_api_token` | Specifies the GitHub API token for accessing private repositories |
| `ssh_private_key` | Specifies the SSH private key (file path or base64 encoded) for accessing private repositories |
| `init-file` | Specifies the location of the init file (local path or `https://` URL) |

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

repository:
  gh_api_token: your_github_api_token
  ssh_private_key: your_ssh_private_key_base64
  init-file: https://example.com/init.yaml

log:
  level: info
```

RWR reads only the keys listed above.

## Modifying the Configuration File

You can modify the `config.yaml` file directly using a text editor. Alternatively, you can use the `rwr config` command to interactively create or update the configuration file.

```bash
rwr config --create
```

This command will prompt you for the necessary settings and generate the `config.yaml` file based on your input.

## Precedence

The settings in the `config.yaml` file have precedence over the default values used by RWR. However, command-line flags, when provided, will override the corresponding settings in the configuration file.

## Environment Variables

RWR reads the same options from the environment. An environment variable takes
precedence over `config.yaml` but is overridden by a command-line flag.

The name is `RWR_` followed by the option's full key in upper case, with each
dot **and each hyphen** replaced by an underscore.

| Config key | Environment variable |
|---|---|
| `log.level` | `RWR_LOG_LEVEL` |
| `repository.gh_api_token` | `RWR_REPOSITORY_GH_API_TOKEN` |
| `repository.ssh_private_key` | `RWR_REPOSITORY_SSH_PRIVATE_KEY` |
| `repository.init-file` | `RWR_REPOSITORY_INIT_FILE` |
| `rwr.configdir` | `RWR_RWR_CONFIGDIR` |
| `rwr.skipVersionCheck` | `RWR_RWR_SKIPVERSIONCHECK` |

```bash
RWR_LOG_LEVEL=debug rwr all
RWR_REPOSITORY_GH_API_TOKEN=your_token rwr all
RWR_REPOSITORY_INIT_FILE=/path/to/init.yaml rwr all
```

> [!NOTE]
> The hyphen in `repository.init-file` becomes an underscore in the environment
> name. `RWR_REPOSITORY_INIT-FILE`, with a literal hyphen, is a different name
> and is ignored.

## Notes

- The `ssh_private_key` in the `repository` section is used as the default SSH key for RWR operations, including private git clones. This key is set when an SSH key is generated with the `set_as_rwr_ssh_key: true` option in the SSH Keys blueprint.
- When using URL sources for files or init files, RWR will download the file from the specified URL before processing it.

For more information on using the configuration file and its options, please refer to the [Commands and Flags](command-and-flags.md) and [Best Practices](../best-practices.md) guides.
