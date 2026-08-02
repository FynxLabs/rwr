# Command Execution — Deltas

## MODIFIED Requirements

### Requirement: Interactive commands get the real terminal

A command with `Interactive` set SHALL receive the real tty (stdin, stdout,
stderr). Under the TUI this SHALL happen via terminal handoff
(`tea.ExecProcess`): the TUI suspends, the child owns the terminal, the TUI
repaints on exit. The handoff SHALL apply to per-item `interactive: true`
inside an otherwise non-interactive run.

Why: piping stderr swallows sudo's password prompt and hangs the run; gating
handoff on the global flag breaks `ResolveInteractive`'s per-item override.

#### Scenario: sudo prompt under the TUI

- **GIVEN** a `files` blueprint requiring elevation
- **WHEN** sudo prompts for a password mid-run
- **THEN** the prompt appears on the real terminal, the user answers, and the
  TUI repaints

#### Scenario: Per-item interactive in a non-interactive run

- **GIVEN** `--interactive=false` and one package with `interactive: true`
- **WHEN** that package installs
- **THEN** the terminal is handed off for that command only

## ADDED Requirements

### Requirement: Non-interactive command output is attributed per line

Stdout and stderr of non-interactive commands SHALL be captured line-wise,
attributed to the running processor, and marked by source (stdout/stderr).
The full stderr text SHALL remain available to the error path, and
per-blueprint `logName` files SHALL keep working.

#### Scenario: Attribution in the store

- **GIVEN** a package manager writing progress to stdout during `packages`
- **WHEN** the lines are captured
- **THEN** each record carries processor `packages` and source `stdout`
