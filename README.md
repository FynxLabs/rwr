# Rinse, Wash, Repeat (RWR)

[![Build](https://github.com/FynxLabs/rwr/actions/workflows/go.yml/badge.svg)](https://github.com/FynxLabs/rwr/actions/workflows/go.yml)

![RWR Logo](img/rwr_128.gif)

RWR rebuilds a workstation from a version-controlled set of blueprints. Describe
the packages, files, repositories, services, accounts, and desktop settings you
want; RWR validates that description and applies it on Linux, macOS, or Windows.

It is built for people who reinstall often, move between machines, or
want their setup recorded somewhere better than shell history.

## What RWR gives you

- **One repeatable setup.** Keep machine configuration in Git and run it again
  after a reinstall or on a new computer.
- **Small, focused blueprints.** Manage packages, files, services, Git checkouts,
  users, SSH keys, fonts, system settings, and Omarchy desktop configuration.
- **Profiles without separate trees.** Share a common base and select additions
  such as `work`, `gaming`, or `laptop` at run time.
- **Safe inspection.** Validate a tree and preview a run before changing the
  machine.
- **Run records.** Inspect drift with `rwr status` and reverse supported recorded
  changes with `rwr uninstall`.
- **Credential setup on your schedule.** Finish ordinary provisioning first,
  then configure Bitwarden, the OS keyring, or GPG tasks explicitly.

Blueprints can be written in YAML, JSON, TOML, or CUE. A repository may contain
one setup or several machine-specific configurations.

## Install

On Linux or macOS:

```bash
curl -sSL https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.sh | sudo bash
```

On Windows, open PowerShell as an administrator:

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.ps1'))
```

The installers select the correct release and verify its checksum. You can also
download an archive or native package from the
[releases page](https://github.com/FynxLabs/rwr/releases/latest). Read the
[installation guide](docs/install.md) for package formats, supported
architectures, nightly builds, and pinned versions.

Review any script before running it with administrator privileges.

## Your first blueprint

Create this small directory:

```text
my-blueprints/
├── init.yaml
└── files/
    └── welcome.yaml
```

`init.yaml` tells RWR where the blueprints live:

```yaml
blueprints:
  format: yaml
  location: .
```

`files/welcome.yaml` contains one desired change:

```yaml
files:
  - name: rwr-was-here.txt
    action: create
    target: "{{ .User.home }}"
    content: |
      This file was created by RWR.
```

Validate it, preview it, then apply it:

```bash
rwr validate ./my-blueprints
rwr all --init-file ./my-blueprints/init.yaml --dry-run
rwr all --init-file ./my-blueprints/init.yaml
```

Once you have a repository you use regularly, `rwr config create` can save its
init-file location so you do not need to pass `--init-file` on every run.

The [quick-start guide](docs/quick-start.md) continues from this example. If the
machine is already configured the way you like, start with
[`rwr capture`](docs/cli/capture.md) instead.

## How a setup is organized

RWR uses three layers:

1. The optional local `config.yaml` remembers where your setup lives and holds
   local CLI preferences.
2. The repository's `init.yaml` selects the blueprint directory, execution
   order, profiles, package managers, and credential policy.
3. Blueprint files describe the actual resources RWR should manage.

Blueprint filenames and folders are mainly for people. RWR can route files by
their top-level keys, so a small setup may use one file while a larger setup can
split work by operating system, machine, or resource type.

## Everyday commands

| Command | Purpose |
|---|---|
| `rwr validate PATH` | Check a blueprint tree without applying it |
| `rwr all` | Apply the selected tree |
| `rwr all --dry-run` | Show the planned work without changing the machine |
| `rwr run packages` | Run one processor; `rwr packages` is equivalent |
| `rwr all --profile work` | Apply base items plus matching profile items |
| `rwr run all --except credentials` | Provision without credential setup |
| `rwr run credentials --profile bitwarden` | Run explicit credential setup |
| `rwr run configuration` | Apply system and desktop configuration, including Omarchy |
| `rwr status` | Compare the desired tree with the current machine |
| `rwr uninstall` | Reverse supported changes from the run record |

Profiles are opt-in: `rwr all` without `--profile` applies only unprofiled
entries, and naming a profile adds its entries on top (`--profile all` applies
everything). See [Profiles](docs/profiles.md) before using profiles as an
environment boundary.

## What can go in a blueprint?

| Area | Blueprint support |
|---|---|
| Software | Package repositories and packages across supported package managers |
| Files | Files, templates, directories, permissions, and remote sources |
| System | Services, users, groups, SSH keys, fonts, and platform settings |
| Development | Git repositories and scripts for work outside a native processor |
| Desktop | dconf/gsettings, macOS defaults, Windows Registry, and Omarchy resources |
| Credentials | Explicit provider setup, scoped resource dependencies, keyring and GPG tasks |

RWR keeps native operations structured and uses scripts as an escape hatch. It
does not hide the fact that a blueprint can install software, edit files, or run
commands: review a blueprint repository before applying it.

## Find the right documentation

If you are new, use these in order:

1. [Quick Start](docs/quick-start.md)
2. [How blueprints work](docs/blueprints-general.md)
3. [Init file reference](docs/init-file.md)
4. [Blueprint type reference](docs/blueprints/README.md)

For a specific task:

- [Commands and flags](docs/cli/command-and-flags.md)
- [Profiles](docs/profiles.md)
- [Variables and templates](docs/variables.md)
- [Credentials and Bitwarden](docs/credentials.md)
- [Omarchy configuration tool](docs/blueprints/omarchy.md)
- [Run records, status, and uninstall](docs/state.md)
- [Examples](examples/README.md)

The [documentation index](docs/README.md) groups the remaining guides by goal.

## Development

RWR uses [mise](https://mise.jdx.dev/) to install its development tools and run
common tasks:

```bash
git clone https://github.com/FynxLabs/rwr.git
cd rwr
mise install
mise run build
mise run ci
```

The test, lint, and security checks also run on pull requests. Behavioral
changes should update the matching living specification under `openspec/`.

Issues and pull requests are welcome. Please include tests for behavior changes
and keep documentation focused on behavior that exists today.

## License

RWR is available under the [MIT License](LICENSE).
