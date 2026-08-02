# Configuration File

The Rinse, Wash, Repeat (RWR) configuration file (`config.yaml`) stores the settings for the RWR tool. This page describes the structure of the configuration file and the settings that are available in it.

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

* A leading `~` is expanded.
* RWR creates the configuration directory and its `run_once` subdirectory if they do not exist.
* A missing configuration file is not an error: RWR logs it at debug level and continues with the defaults.

## File Format

The configuration file uses the YAML format. It consists of key-value pairs and nested sections to organize the settings.

## Configuration Settings

The following settings are available in the `config.yaml` file:

### `rwr` Section

The `rwr` section contains general settings for the RWR tool.

| Setting | Description |
|---------|-------------|
| `configdir` | The directory that `rwr config --create` writes `config.yaml` to, and the directory that holds the `bootstrap` marker file. The default is `$HOME/.config/rwr`. It does **not** change where RWR reads `config.yaml` from — use `--config` for that |

> [!NOTE]
> There is no working `rwr.skipVersionCheck` option. The key is written by
> `rwr config --create`, but nothing reads it back: only the
> `--skip-version-check` flag turns the version check off.

### `repository` Section

The `repository` section contains settings related to Git repositories.

| Setting | Description |
|---------|-------------|
| `gh_api_token` | Specifies the GitHub API token for accessing private repositories |
| `ssh_private_key` | Specifies the SSH private key (file path or base64 encoded) for accessing private repositories |
| `init-file` | Specifies the location of the init file (local path or `https://` URL) |

### `log` Section

The `log` section contains settings related to logging.

| Setting | Description |
|---------|-------------|
| `level` | Specifies the log level (debug, info, warn, error) |

## Example Configuration File

Here is an example `config.yaml` file:

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

RWR reads only the keys listed above. `rwr config --create` also writes
`rwr.skipVersionCheck`, which nothing reads.

## Modifying the Configuration File

You can modify the `config.yaml` file directly with a text editor. You can also use the `rwr config` command to interactively create or update the configuration file.

```bash
rwr config --create
```

This command prompts you for the settings. Then it writes the `config.yaml` file.

## Precedence

The settings in the `config.yaml` file have precedence over the default values used by RWR. However, a command-line flag overrides the corresponding setting in the configuration file.

## Environment Variables

RWR reads the same settings from the environment. An environment variable takes
precedence over `config.yaml`, but a command-line flag overrides it.

The name is `RWR_` followed by the full key of the setting in upper case, with each dot
replaced by an underscore. **A hyphen in the key is kept as a hyphen** — it is
not turned into an underscore.

| Config key | Environment variable |
|---|---|
| `log.level` | `RWR_LOG_LEVEL` |
| `repository.gh_api_token` | `RWR_REPOSITORY_GH_API_TOKEN` |
| `repository.ssh_private_key` | `RWR_REPOSITORY_SSH_PRIVATE_KEY` |
| `repository.init-file` | `RWR_REPOSITORY_INIT-FILE` |
| `rwr.configdir` | `RWR_RWR_CONFIGDIR` |

```bash
RWR_LOG_LEVEL=debug rwr all
RWR_REPOSITORY_GH_API_TOKEN=your_token rwr all
```

> [!NOTE]
> `RWR_REPOSITORY_INIT-FILE` contains a hyphen, so most shells will not accept it
> in the `NAME=value command` form. Use `env` instead, or pass `--init-file`:
>
> ```bash
> env "RWR_REPOSITORY_INIT-FILE=/path/to/init.yaml" rwr all
> ```
>
> `RWR_REPOSITORY_INIT_FILE`, with an underscore, is a different name and is
> ignored.

## Notes

- The `ssh_private_key` in the `repository` section is used as the default SSH key for RWR operations, including private git clones. RWR sets this key when it generates an SSH key with the `set_as_rwr_ssh_key: true` setting in the SSH Keys blueprint.
- When a file source or init file is a URL, RWR downloads the file before it processes it.

For more information on the configuration file and its settings, read the [Commands and Flags](command-and-flags.md) and [Best Practices](../best-practices.md) guides.
