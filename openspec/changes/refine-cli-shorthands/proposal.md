# Change: CLI polish — config view/edit, shorthand rationalization

## Why

Two leftover roughnesses after the fossil-kill batch (#209 landed the rest):

**`rwr config` is create-only.** The command's own help admits it: "there is
no config view or edit mode yet" (`cmd/config.go:16-18`). `--create`/`-c` is
the single flag (`cmd/config.go:34`); anything else prints help. There is no
way to see the effective config (which matters — it merges file, env, and
flags via viper) or to open it in an editor, and the config file can contain
the GitHub token, so "just cat it" is the wrong answer to teach.

**The shorthand table is sparse and lopsided.** Current shorts on root
persistent flags (`cmd/root.go:121-186`): `-d` debug (`:125`), `-I`
interactive (`:131`), `-i` init-file (`:138`), `-p` profile (`:181`). That is
the complete list. The two flags people type most in day-to-day use have no
short: `--dry-run` (`:128`, plus its `--no-op` alias `:129`) and `--log-level`
(`:126`). Meanwhile `-i`/`-I` is a trap: the same letter in two cases means
two unrelated things ("init file" vs "interactive"), and `-I` is a bool
defaulting to *true*, so `-I` alone is a no-op and the useful spelling is
`--interactive=false`.

## What Changes

- **`rwr config` grows subcommands** (bare `rwr config` keeps printing help):
  - `rwr config view` — the effective merged config, secret values shown as
    `[redacted]` (reusing `types.RedactedPlaceholder`; `--show-secrets`
    reveals, matching the existing log convention at `cmd/root.go:178`), each
    key annotated with its source (default/file/env/flag) where viper can say.
  - `rwr config edit` — open the config file in `$EDITOR` (fallback
    `$VISUAL`, then a platform default), creating it via the existing
    `CreateDefaultConfig` path if absent; re-validate after the editor exits
    and warn on parse errors rather than leaving them to the next run.
  - `rwr config create` — subcommand form of today's `--create`. The
    `--create`/`-c` flag stays as a deprecated alias (cobra
    `MarkDeprecated`, the `--gh-key` pattern at `cmd/root.go:150-154`).
- **New shorthands, additive only**:
  - `-n` for `--dry-run` (the `make -n` convention; `--no-op` alias remains
    flagged-only)
  - `-l` for `--log-level`
- **`-i`/`-I` stays as-is, documented as a decision.** Reassigning either
  letter breaks existing scripts and muscle memory for marginal gain. Instead:
  help text for the pair cross-references the other, and `--interactive`
  gains the explicit guidance that `--interactive=false` is the operative
  spelling. If a future breaking window opens (e.g. a 2.0), revisit.
- Verify no collisions: `-n`, `-l`, and `-c` (config-local) are unclaimed
  today across root and subcommand flag sets.

## Breakage

Nothing breaks. All new shorts are additions on previously short-less flags;
`rwr config --create` keeps working (deprecated, with a message naming
`rwr config create`); no existing flag, short, default, or command is removed
or re-meaninged.

## Impact

- Affected specs: `cli`.
- Affected code: `cmd/config.go` (subcommands), `cmd/root.go` (two
  `BoolVarP`/`StringVarP` swaps), docs `commands/config.md` and the flag
  tables.
- `config view` must go through the redaction path — it is a new place a
  secret could leak; the test for that is the load-bearing one.
