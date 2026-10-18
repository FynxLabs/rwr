# Command Execution Specification

## Purpose

RWR configures a machine by running other programs - package managers, `systemctl`,
`ssh-keygen`, `useradd`, operator scripts. Every one of those invocations carries
values that came out of a blueprint file, and blueprints are cloned from git
repositories. This capability defines how a `types.Command` becomes a real process,
how elevation is applied, how output is captured, and how the whole path is made
observable in tests without spawning anything.

## Requirements

### Requirement: Operator interaction stays inside the dashboard

While the dashboard is active, RWR SHALL retain terminal ownership. Forms,
credential entry, confirmations, and asynchronous authentication instructions
SHALL use shared modal APIs. An interactive external command SHALL receive a
private pseudoterminal rendered inside a modal, with keyboard input, paste,
terminal sizing, and password echo behavior preserved. RWR SHALL NOT suspend
the dashboard to let a prompt or tool own the real terminal.

Known secrets SHALL be redacted from tool output across write boundaries.
Authentication output marked private SHALL be displayed without entering run
logs. Cancellation or loss of the dashboard SHALL stop the modal's background
work and child process; Unix cancellation SHALL stop the child process group.
Headless invocations SHALL retain direct terminal interaction.

Visible masked credential prompts SHALL own keyboard and paste input ahead of
underlying forms. Completing an overlay or asynchronous form SHALL preserve any
other pending interaction instead of returning the dashboard to running state.

Pre-supplied interactive input SHALL be staged without echo before starting the
child or forwarding operator input. The child SHALL retain a TTY on stdin and
control echo for subsequent input. RWR SHALL refuse input that cannot be queued
intact. Canceling an operator choice SHALL propagate cancellation, never approve
its default action.

#### Scenario: An external tool asks for MFA

- **WHEN** an interactive credential tool requests a verification code
- **THEN** its instructions and input appear in a modal terminal pane
- **AND** the dashboard remains visible and owns the terminal

#### Scenario: A tool popup is canceled

- **WHEN** the operator presses Ctrl-C in the external-tool popup
- **THEN** the tool is stopped and the dashboard remains active

#### Scenario: Sudo and GitHub authentication overlap

- **WHEN** a masked sudo prompt and a GitHub form are both pending
- **THEN** the visible sudo prompt receives keyboard and paste input
- **AND** completing it restores the GitHub form if it is still waiting
- **AND** asynchronous GitHub completion does not hide the sudo prompt

#### Scenario: A raw terminal handoff is attempted

- **WHEN** code requests a raw terminal callback while the dashboard is active
- **THEN** the callback is refused without releasing the terminal

#### Scenario: Initial input is followed by a password prompt

- **WHEN** an interactive command receives pre-supplied input and later disables
  echo through stdin to request a new password
- **THEN** neither the initial input nor that password appears in the popup or log

#### Scenario: The package-manager chooser is canceled

- **WHEN** an operator cancels the chooser on a fresh Mac
- **THEN** RWR returns cancellation without installing the default package manager

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

On Windows, `Elevated` SHALL be a no-op at this layer: there is no `sudo`
equivalent. A processor that genuinely needs elevation on Windows SHALL raise it
itself through `Start-Process -Verb RunAs`, and SHALL pass the elevated process
its inputs as data rather than as command text - the elevated shell tokenizes its
argument list a second time, so any value interpolated into it becomes code
running as administrator. The Windows registry processor does this by writing a
JSON payload and invoking a constant script via `-EncodedCommand`.

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
- **THEN** RWR collects the password through its managed masked popup
- **AND** the command receives its popup terminal for its own interaction

### Requirement: Sudo readiness checks never prompt or delay cancellation

Before running a command that is elevated, runs as another user, or may invoke
`sudo` itself, RWR SHALL check the sudo credential cache using exactly
`sudo -n -v`. The check SHALL NOT prompt for a password.

The check SHALL use the active run context and a bounded timeout. Cancelling the
run SHALL cancel an in-flight check immediately. RWR SHALL cache both a warm and
a cold result for one minute so a stalled policy backend cannot impose its full
timeout once per package.

A cached cold result SHALL NOT be treated as warm. Before a command that may
invoke sudo, RWR SHALL collect a password through a masked prompt inside an
active TUI. When no TUI is active, RWR SHALL use the masked terminal fallback.
It SHALL validate the password with `sudo -S -v`. The password SHALL NOT appear
in argv, logs, or terminal output and SHALL be cleared from its temporary byte
buffers after use. The actual command SHALL remain captured by the TUI after
validation; collecting the password SHALL NOT suspend or replace the dashboard.

A provider-wide escalation declaration SHALL be narrowed when the operation can
be classified: Homebrew formulae remain captured without sudo validation, while
casks retain the secure validation step.

#### Scenario: Sudo credentials are not cached

- **WHEN** `sudo -n -v` reports that credentials are not cached
- **THEN** RWR asks once through the TUI's masked prompt, or the masked terminal fallback when headless
- **AND** validates the supplied password without placing it in argv
- **AND** runs the actual command captured by the TUI

#### Scenario: A warm probe does not cover the command's authorization

- **WHEN** the readiness probe succeeds but an elevated command cannot reuse
  that authorization in its session, TTY, or policy
- **THEN** that sudo process requests the masked password popup through askpass
- **AND** captured, interactive, and output-returning execution can authenticate
- **AND** the command's supplied stdin remains available to the command
- **AND** a valid ticket or password-free command runs without an extra prompt
- **AND** canceling password entry stops the waiting command

#### Scenario: A Homebrew formula with a cold sudo cache

- **WHEN** Homebrew metadata classifies a package operation as a formula
- **THEN** the command remains captured by the TUI
- **AND** the dashboard is not suspended for that package

#### Scenario: The sudo policy backend stalls

- **WHEN** the non-interactive sudo check reaches its timeout
- **THEN** the result is cached briefly as cold
- **AND** adjacent package commands do not each repeat the stalled check
- **AND** the secure validation prompt is used before the next command that may
  invoke sudo

#### Scenario: No terminal is available

- **WHEN** sudo credentials are cold and stdin is not a terminal
- **THEN** RWR does not attempt to read a password
- **AND** the command proceeds so command-scoped `NOPASSWD` policies still work

#### Scenario: The operator cancels during the sudo check

- **WHEN** the operator cancels the run while the sudo check is blocked
- **THEN** the check stops through the run context without waiting for its
  fallback timeout

### Requirement: A command's log file stays open while the command runs

When a command declares `LogName` and debug output is off, RWR SHALL open that file
and point the command's stdout at it. RWR SHALL close the file after the command
finishes, not before.

The log path comes from the blueprint (`log:` on a script), so it is
attacker-influenced whenever the blueprint is. RWR SHALL refuse to write a command
log through a symlink, and SHALL create the log file at `0600`.

Appending through a symlink would put command output into `~/.ssh/authorized_keys`
or `~/.bashrc` without the blueprint containing anything that looks like it writes
a file, and command output routinely contains more than the operator expects.

#### Scenario: A script that writes to its declared log

- **WHEN** a blueprint declares `log: build.log` for a command that writes output
- **THEN** `build.log` contains what the command wrote
- **AND** the file is created at `0600`

#### Scenario: A log path that is a symlink

- **WHEN** a blueprint declares `log:` at a path that is a symlink
- **THEN** RWR refuses the command with an error naming the path, and writes nothing
  through the link

### Requirement: A secret reaches a tool on standard input, not in argv

`types.Command` SHALL carry a `Stdin` field, and RWR SHALL feed its contents to the
spawned process on standard input. `Stdin` SHALL NOT be a blueprint field: it exists
so RWR can hand a tool a credential without putting it in argv.

Where a tool accepts a credential on stdin, RWR SHALL use that path. Setting a Linux
account password SHALL go through `chpasswd` on stdin - `chpasswd -e` when the value
is already a crypt(3) hash, plain `chpasswd` otherwise - and SHALL NOT use
`useradd`/`usermod --password`.

An argument of a `sudo`'d process is readable in `ps` by every local user for the
lifetime of the call, lands in sudo's syslog record, and is printed verbatim by the
debug logger, which logs `cmd.Args`. `usermod --password` is wrong a second way: it
writes its argument into the hash field of `/etc/shadow`, so a cleartext value
becomes a hash nothing can match and the account silently has no password.

Supplied input SHALL win over the terminal even for an interactive command, because
inheriting `os.Stdin` would hang waiting for a human who has nothing to type.

#### Scenario: Setting a Linux user password

- **WHEN** a users blueprint declares a cleartext password
- **THEN** `chpasswd` is spawned with `<name>:<password>` on stdin
- **AND** the password does not appear in the argv of any process

#### Scenario: A pre-hashed password

- **WHEN** the declared password is already a crypt(3) hash such as `$6$...`
- **THEN** `chpasswd -e` is used so the hash is stored as given

### Requirement: Values a tool only accepts as an argument are redacted from logs

`types.Command` SHALL carry a `Secrets` field listing values that must not be
written to a log, and SHALL expose a `LogArgs()` method returning `Args` with every
such value replaced by the redaction placeholder. Every log line that prints a
command's arguments - the debug lines in `buildCommand`, and the `[DRY-RUN]` line -
SHALL use `LogArgs()` rather than `Args`.

Some tools accept a credential only as an argument (chocolatey's `--password`,
cargo's login token), so the value cannot always be moved to `Stdin`. It should
still never be written to a log file.

#### Scenario: A command carrying a credential in its arguments

- **WHEN** a command lists its token in `Secrets` and debug logging is on
- **THEN** the logged argument list shows the redaction placeholder in place of the
  token

#### Scenario: A command with no declared secrets

- **WHEN** `Secrets` is empty
- **THEN** `LogArgs()` returns the arguments unchanged

### Requirement: A script may declare the account it runs as

`types.Script` SHALL accept an `asUser` field, which RWR SHALL map to the command's
`AsUser` and therefore to `sudo -u`.

When a script declares both `elevated` and `asUser`, RWR SHALL run it elevated,
SHALL ignore `asUser`, and SHALL warn naming the ignored account. `sudo` cannot do
both at once and command construction checks `Elevated` first, so the previous
behavior silently dropped one of the two.

#### Scenario: A script run as another account

- **WHEN** a script declares `asUser: levi` and does not declare `elevated`
- **THEN** the script is spawned as `sudo -u levi -- <interpreter> <script>`

#### Scenario: A script declaring both

- **WHEN** a script declares `elevated: true` and `asUser: levi`
- **THEN** the script runs elevated
- **AND** RWR warns that `asUser: levi` was ignored

### Requirement: Dry-run reports commands without running them

When dry-run mode is on, RWR SHALL log each command it would execute and SHALL NOT
spawn a process. RWR SHALL report dry-run mode at the start of the run and summarize
at the end.

`--dry-run` and `--no-op` SHALL select the same behavior.

#### Scenario: A full run in dry-run mode

- **WHEN** `rwr all --dry-run` runs against a blueprint tree
- **THEN** every command is logged with a `[DRY-RUN]` marker
- **AND** no package is installed, no file is written, and no service changes state

#### Scenario: A dry-run of a command carrying a secret

- **WHEN** a command listing a value in `Secrets` is reported in dry-run mode
- **THEN** the `[DRY-RUN]` line shows the redaction placeholder, not the value

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

### Requirement: Installer steps stage in a private per-run directory

RWR SHALL create one private staging directory per run - a `0700` temporary
directory with an unpredictable name, reachable only by the invoking user - and
SHALL render `{{ .TempDir }}` in package-manager install and remove steps to
it. When the directory cannot be created, the run SHALL stop; there is no
fallback path.

Install steps historically staged at fixed, world-known `/tmp` names
(`/tmp/brew-install.sh`, a clone into `/tmp/yay`), and the macports installer
staged its download in the current working directory. Any local user can
pre-create such a path - or rewrite it between the download and the elevated
step that executes it - which is root code execution. A failed creation used to
fall back to the predictable `<tmp>/rwr-pm-unavailable`, which is exactly the
class of name `{{ .TempDir }}` exists to eliminate.

#### Scenario: A provider step referencing the staging directory

- **WHEN** an install step names `{{ .TempDir }}/installer.pkg` as its download destination
- **THEN** the path renders inside the run's private staging directory
- **AND** a later step in the same run renders the same directory

#### Scenario: The staging directory cannot be created

- **WHEN** creating the temporary directory fails
- **THEN** the run stops with an error
- **AND** no fixed fallback path is used

### Requirement: Captured nested sudo uses managed prompts

On Linux and macOS, captured commands marked `Escalates` SHALL use a
command-scoped askpass bridge while an inline display owns the terminal.
Elevated commands whose readiness probe succeeded SHALL retain managed askpass
so a different child session or TTY cannot suppress required password entry.
Interactive popup processes SHALL bind the bridge to their actual child session
after startup, with the same peer validation as captured commands.
Helpers SHALL be authenticated using kernel-reported user identity and the
command's isolated process session. Knowing the endpoint path SHALL NOT grant
access from another session. RWR SHALL NOT export an authentication token to
children or persist one on disk.

Password bytes SHALL travel over the private socket and the askpass helper's
stdout as consumed by sudo. The socket SHALL be owner-readable/writable, inside
an owner-only directory. Commands explicitly marked `Interactive` SHALL retain
the terminal handoff. A command failure SHALL NOT automatically retry with
terminal access.

#### Scenario: A child sudo cannot reuse the parent's sudo ticket

- **WHEN** an unprivileged AUR helper calls sudo from a captured command
- **AND** sudo needs authentication despite an earlier successful validation
- **THEN** sudo obtains the password through the managed askpass bridge
- **AND** no raw terminal password prompt overwrites the dashboard

#### Scenario: Another process discovers the endpoint

- **WHEN** a process outside the command's session invokes the askpass helper
- **THEN** the bridge rejects it before prompting or returning password bytes

#### Scenario: Authentication is rejected

- **WHEN** sudo rejects a password
- **THEN** RWR allows up to three masked validation attempts
- **AND** never changes a captured command to interactive as a retry strategy

#### Scenario: A pending authentication request is cancelled

- **WHEN** a command finishes or the operator cancels the run
- **THEN** its pending secret request, connections, and listener are closed
- **AND** its private bridge directory is removed
- **AND** whole-run cancellation is reported as `ErrCancelled`

### Requirement: Provider escalation policy reaches every command path

Provider `Escalates` and `Elevated` settings SHALL be preserved for package
commands, startup dependencies, repository commands, updates, and cleanup.
Install/remove command steps MAY override either setting explicitly.

#### Scenario: Manifest selection requires a missing build dependency

- **WHEN** a selected configuration declares an AUR helper
- **AND** preparation installs a missing build dependency before bootstrap
- **THEN** that dependency command carries the helper's self-escalation policy
- **AND** nested sudo receives the managed password prompt

#### Scenario: Installing an AUR helper

- **WHEN** its prerequisite pacman step runs
- **THEN** RWR elevates that step
- **AND** git clone runs unprivileged without requesting sudo
- **AND** makepkg builds as the user and uses managed nested sudo for installation


## Known Gaps

- **The macOS account password is still passed in argv.** `dscl . -passwd` takes
  the cleartext password as an argument, so it is visible in `ps` to every local
  user and recorded by sudo. RWR warns when it does this, and lists the value in
  `Secrets` so it is at least kept out of the logs. Open Directory computes its own
  salted hash, so there is no pre-computed value to send instead; `dscl` offers no
  stdin equivalent, so nothing keeps it out of `ps`.
- **Repository credentials are still passed in argv.** chocolatey's `--password`
  and cargo's login token are accepted only as arguments. They are listed in
  `Secrets`, so they do not reach the logs, but they are readable in `ps` while the
  command runs.
- **Very short secrets are not redacted.** `LogArgs` skips any secret under four
  characters, because substring-replacing one would corrupt unrelated arguments.
