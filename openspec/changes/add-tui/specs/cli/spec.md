# CLI - Deltas

## ADDED Requirements

### Requirement: TUI activates only on a real TTY

`rwr all` and `rwr run` SHALL render the TUI only when stdout is a terminal
and none of `--no-tui`, `CI`, or `TERM=dumb` apply. In every other case output
SHALL be byte-identical to the pre-TUI streaming logs.

Why: escape sequences in a CI build log or a redirected file are corruption,
not UI.

#### Scenario: Redirected output unchanged

- **GIVEN** `rwr all > install.log`
- **WHEN** the run completes
- **THEN** `install.log` matches the pre-TUI output byte for byte

#### Scenario: CI runner with a pty

- **GIVEN** `CI=true` on a runner that allocates a pty
- **WHEN** `rwr all` runs
- **THEN** no escape sequences are emitted

### Requirement: Run summary and run log

A TUI run SHALL record every log line to a run log file (mode 0600, path
printed on exit, `--log-file` overrides) and SHALL end with a summary of
resources by status: applied, skipped, failed, unknown.

Why: `unknown` is required - several providers do not report per-package
results reliably; forcing them into ok/failed fabricates data.

#### Scenario: Failure visible at end

- **GIVEN** a 13-processor run where processor 1 failed
- **WHEN** the run reaches the summary
- **THEN** the failure is on screen without scrolling, and the exit code is
  nonzero
