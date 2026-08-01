# Command Execution Specification

## Purpose

RWR configures a machine by running other programs — package managers, `systemctl`,
`ssh-keygen`, `useradd`, operator scripts. Every one of those invocations carries
values that came out of a blueprint file, and blueprints are cloned from git
repositories. This capability defines how a `types.Command` becomes a real process,
how elevation is applied, how output is captured, and how the whole path is made
observable in tests without spawning anything.

## Requirements

### Requirement: Commands execute as argv, never as a shell string

RWR SHALL build every command as an argument vector handed directly to the kernel.
RWR SHALL NOT interpose a shell, and SHALL NOT construct a command by concatenating
strings.

Each element of `Command.Args` SHALL reach the target program as exactly one
argument. A blueprint SHALL be able to supply a value containing shell
metacharacters, spaces, or newlines without that value being reinterpreted.

A provider that genuinely needs a shell SHALL request one explicitly, by naming the
shell as `Exec` and passing the script as an argument.

#### Scenario: A package name containing shell syntax

- **WHEN** a blueprint declares a package named `vim; touch /tmp/pwned`
- **THEN** the spawned process receives `install` and `vim; touch /tmp/pwned` as two
  discrete arguments
- **AND** no file `/tmp/pwned` is created

#### Scenario: An argument containing spaces

- **WHEN** a command is built with the argument `Some Name`
- **THEN** the target program receives `Some Name` as a single argument, not two

#### Scenario: A provider that wants a shell

- **WHEN** a provider declares `exec = "sh"` with `args = ["-c", "cd /tmp/paru && makepkg"]`
- **THEN** the shell runs and interprets the script, because the provider asked for it

### Requirement: Elevation is per-command and terminates option parsing

RWR SHALL apply elevation per command, from the command's own `Elevated` and
`AsUser` fields. RWR SHALL NOT acquire elevation for the whole run.

On a non-Windows host:

- `Elevated: true` SHALL produce `sudo -- <exec> <args...>`.
- `AsUser: "<name>"` SHALL produce `sudo -u <name> -- <exec> <args...>`.
- Neither set SHALL produce `<exec> <args...>` with no `sudo`.

The `--` separator SHALL be present so a program whose name begins with a dash
cannot be absorbed as a `sudo` option.

On Windows, `Elevated` SHALL be a no-op: there is no `sudo` equivalent, and the
process must already be elevated.

#### Scenario: An elevated command on Linux

- **WHEN** a command with `Elevated: true` runs `pacman` with `-S git`
- **THEN** the argv is `sudo -- pacman -S git`

#### Scenario: A command run as another user

- **WHEN** a command with `AsUser: "levi"` runs `paru` with `-S foo`
- **THEN** the argv is `sudo -u levi -- paru -S foo`

#### Scenario: An unelevated command

- **WHEN** a command with `Elevated: false` and no `AsUser` runs
- **THEN** the argv contains no `sudo` and no shell

#### Scenario: Elevation on Windows

- **WHEN** `GOOS` is `windows` and a command declares `Elevated: true`
- **THEN** the program is spawned directly
- **AND** the argv contains no `sh`, no `cmd /C`, and no `sudo`

### Requirement: Interactive commands reach the terminal

RWR SHALL connect an interactive command's stdin, stdout, and stderr to the
terminal. RWR SHALL NOT buffer stderr for an interactive command.

This exists because `sudo` writes its password prompt to stderr. Capturing it
leaves the run blocked with no indication of what it waits for.

#### Scenario: A command that prompts for a password

- **WHEN** an interactive elevated command runs and `sudo` requires a password
- **THEN** the prompt appears on the terminal and the operator can answer it

### Requirement: A command's log file stays open while the command runs

When a command declares `LogName` and debug output is off, RWR SHALL open that file
and point the command's stdout at it. RWR SHALL close the file after the command
finishes, not before.

#### Scenario: A script that writes to its declared log

- **WHEN** a blueprint declares `log: build.log` for a command that writes output
- **THEN** `build.log` contains what the command wrote

### Requirement: Dry-run reports commands without running them

When dry-run mode is on, RWR SHALL log each command it would execute and SHALL NOT
spawn a process. RWR SHALL report dry-run mode at the start of the run and summarize
at the end.

`--dry-run` and `--no-op` SHALL select the same behavior.

#### Scenario: A full run in dry-run mode

- **WHEN** `rwr all --dry-run` runs against a blueprint tree
- **THEN** every command is logged with a `[DRY-RUN]` marker
- **AND** no package is installed, no file is written, and no service changes state

### Requirement: Command execution is replaceable for tests

RWR SHALL route every command through an `Executor` interface, and SHALL provide a
way to substitute a different implementation and restore the previous one.

A test SHALL be able to run a real processor against real fixture blueprints and
assert on the exact argv, elevation, target user, log name, and variables that the
processor produced, without any process being spawned.

#### Scenario: A processor under test

- **WHEN** a test installs a recording executor and invokes a real processor
- **THEN** every command the processor built is recorded with its exec, args,
  elevation, target user, log name, and variables
- **AND** no package manager runs

### Requirement: The command environment carries no credentials

RWR SHALL copy the current process environment into each spawned command, append
the command's own variables, and extend `PATH` with the common binary directories.

The GitHub API token and the SSH private key SHALL NOT be in that environment unless
the init file opted into them by name. See the credential-handling specification.

#### Scenario: A script spawned by a blueprint

- **WHEN** a blueprint runs a script and the init file names no exposed credentials
- **THEN** `RWR_VAR_REPOSITORY_GH_API_TOKEN` and `RWR_VAR_REPOSITORY_SSH_PRIVATE_KEY`
  are absent from the script's environment
