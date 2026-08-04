# Change: Bubble Tea TUI for interactive runs

## Why

`rwr all` streams logs; on a real provisioning run (13 processors, hundreds of
resources, sudo prompts mid-stream) there is no view of overall progress, no
persistent failure surface, and errors scroll away. A TTY-only dashboard fixes
this; non-TTY behavior stays byte-identical.

## What Changes

Full design in `design.md`. Summary:

- Resolve phase producing a `Plan` (two stages; stage 2 after bootstrap because
  bootstrap can install the package manager later blueprints depend on).
  `rwr validate` consumes stage 1.
- Error accumulation in `All()`: fatal pre-loop errors abort; processor errors
  are collected in non-interactive mode, halt-and-prompt in interactive mode.
  Exit nonzero if any collected.
- `Reporter` event bus; `TUIReporter` and `LogReporter` (byte-identical to
  today's output). Log capture via a `slog.Handler` stamping a package-level
  `currentProcessor` - zero edits to the ten processor files.
- Interactive command handoff via `TerminalReq` + `tea.ExecProcess`; stderr of
  interactive commands is never piped (sudo prompt). Works for per-item
  `interactive: true` inside non-interactive runs.
- Ring-buffered log store with per-processor views, run log file always written.
- Frames: Resolving, Running (header/strip/collapsed/panel/help), Summary,
  Dry-run, Validate. Compact mode below ~60 cols / ~14 rows.
- Theming (11 roles, embedded themes, TOML user themes, `NO_COLOR`/colorprofile
  degradation, ASCII glyph fallback), mouse via bubblezone, animation via
  harmonica, OSC 9 completion notification.
- Activation: TTY check; `--no-tui`, `CI`, `TERM=dumb` fall back to
  `LogReporter`.

Also folds `internal/system/prompt.go` `PromptUserChoice` (raw stdin) into the
reporter/handoff path - a second stdin consumer would fight the TUI.

## Breakage

Nothing breaks. `rwr all > file.log`, `CI=true`, and `TERM=dumb` produce
today's output byte-for-byte (this is a gated test). Blueprint semantics are
untouched.

## Impact

- Affected specs: `cli`, `blueprint-processing`, `command-execution`.
- New direct deps: bubbletea/v2, bubbles/v2, lipgloss/v2, catppuccin/go,
  colorprofile (already in go.sum via huh), harmonica, bubblezone.
- Unblocks: `add-blueprint-manifest` selection frame.
