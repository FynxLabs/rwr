# Commands and Flags

The Rinse, Wash, Repeat (RWR) CLI provides a set of commands and flags to manage your system's configuration. This page describes the available commands and their associated flags.

## Global Flags

The following flags are available for all commands:

| Flag | Description |
|------|-------------|
| `--config` | Path to the config file, or to a directory holding `config.yaml`. The default is `~/.config/rwr`. See [Configuration File](configuration.md) |
| `--debug`, `-d` | Enable debug mode for verbose output |
| `--log-level`, `-l` | Set the log level (debug, info, warn, error) |
| `--init-file`, `-i` | Path to the init file. A directory is accepted: RWR looks in it for `init.yaml`, `init.yml`, `init.json`, `init.toml`, then `init.cue` |
| `--gh-api-key` | Specify the GitHub API key for accessing private repositories |
| `--ssh-key` | Path to an SSH key file, or a Base64-encoded SSH key, for Git authentication |
| `--skip-version-check` | Do not check GitHub for a newer release at startup |
| `--show-secrets` | Print credential values in logs instead of redacting them. Off by default; use it only to confirm RWR is reading the token or key you expect |
| `--dry-run`, `-n` | Simulate operations without making changes (no-op mode) |
| `--no-op` | Alias for `--dry-run` |
| `--interactive`, `-I` | Enable interactive per-item prompts (default: true). Use `--interactive=false` to disable. Capital `-I` — lowercase `-i` is `--init-file` |
| `--gh-key` | Deprecated alias for `--gh-api-key`; use `--gh-api-key` |
| `--gh-auth` | Get a GitHub token with the OAuth device flow. Only `rwr all` and `rwr run ssh_keys` act on it |
| `--profile`, `-p` | Make a profile active. Repeat the flag, or give a comma-separated list |
| `--force-bootstrap` | Run the bootstrap process again |
| `--version`, `-v` | Print the version and exit |

### The startup version check

Unless `--skip-version-check` is given, RWR asks the GitHub releases API for the
latest release when a command starts, and prints one line to stderr if the
binary is older:

```text
rwr: a newer version is available: 0.6.0 (you have 0.5.2)
rwr: https://github.com/fynxlabs/rwr/releases/latest (silence with --skip-version-check)
```

The check is advisory and never fails the run: a two-second timeout, and every
error — no network, a rate limit, an unexpected response — is a debug log and
nothing more. It is also skipped for a development build (one built with plain
`go build`, or from a modified tree), and for `rwr help`, `rwr config`,
`rwr version` and `rwr validate`, which do no version check at all.

> [!NOTE]
> Only the flag turns the check off. There is no configuration-file or
> environment-variable equivalent.

## Commands

The following commands are available in the RWR CLI:

> [!IMPORTANT]
> `rwr` on its own does not apply anything. It prints a greeting and the help
> text. Use `rwr all` to apply every blueprint, or `rwr run <processor>` to apply
> one.

### `rwr version`

Print the build information for this binary.

```console
$ rwr version
rwr 0.5.2
commit:     24872aab978e459254544b1fb58afbb080100cb1
built:      2026-08-01T21:20:55Z
built by:   goreleaser
tree state: dirty
go:         go1.26.5 linux/amd64
```

The `commit`, `built`, `built by` and `tree state` lines are printed only when
the binary carries that information; the `go` line is always printed. A release
or nightly binary carries the commit it was built from, so a binary on disk can
be traced back to its source.

`rwr --version` prints the first line only:

```console
$ rwr --version
rwr 0.5.2
```

A binary built with a plain `go build` reports a development version instead of
a release version, for example
`0.5.2-0.20260801212055-24872aab978e+dirty`, `built by: go`.

### `rwr config`

View, edit, or create the RWR configuration.

| Subcommand | Description |
|------------|-------------|
| `view` | Print the effective merged configuration with per-key sources. Secrets show as `[redacted]`; `--show-secrets` reveals them |
| `edit` | Open the config file in `$VISUAL`/`$EDITOR` (created first if missing; re-parsed after, warning if broken) |
| `create` | Create the configuration file, prompting for each setting |

`--create`/`-c` is deprecated; use `rwr config create`.

### `rwr all`

Run all blueprints and set up the system.

`--force-bootstrap` is a global flag. It also applies to this command.

### `rwr bootstrap`

Run just the bootstrap processor (`rwr run bootstrap` works too). Asking for
bootstrap by name implies wanting it to run, so this ignores the run-once
marker that keeps `rwr all` idempotent — no `--force-bootstrap` needed. The
marker is refreshed on success, `--dry-run` is honored, and a tree without a
bootstrap file is an error naming the filenames searched.
### `rwr status`

Show desired-vs-actual drift without applying anything. Read-only, never
elevates, exits 1 on drift. See [Run records](../state.md).

### `rwr uninstall`

Reverse what recorded runs applied — and only that. Refuses without a run
record; prints the not-reversible list up front; `--yes` skips the prompt.
See [Run records](../state.md).

| Flag | Description |
|------|-------------|
| `--yes` | Skip the confirmation prompt |

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

### `rwr convert`

Convert a blueprint tree between formats, or migrate deprecated constructs.
See [Convert Command](convert.md).

### `rwr profiles`

Read the blueprint tree and list every profile it declares, with the number of
items that carry each one. This command has no flags of its own.

### `rwr run`

Run one processor instead of the whole blueprint. `rwr run` needs the name of a
processor: on its own, or with a name that is not in the list below, it prints
the list and exits with an error.

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
>
> `rwr run` takes one processor, not a list. `rwr run all` runs every
> processor — it is the same run as `rwr all`.
>
> Every processor name also works straight off the root, task-runner style:
> `rwr packages` is `rwr run packages`.

### `rwr completion`

Cobra's generated command. It writes a shell completion script for `bash`,
`zsh`, `fish` or `powershell` to stdout.

## Examples

Here are a few examples of using the RWR CLI with different commands and flags:

```bash
# Apply every blueprint with debug mode enabled
rwr all --debug

# Run the package processor with a specific init file
rwr run packages --init-file path/to/init.yaml

# Create the configuration file
rwr config --create

# Apply every blueprint with bootstrap forced
rwr all --force-bootstrap

# Apply every blueprint, with two profiles active, using a separate config
rwr all --config ~/rwr-work --profile work --profile dev
```

For more detailed information on each command and its usage, please refer to the specific blueprint type documentation or the [Configuration File](configuration.md) page.
