# Tasks

- [ ] 1. `rwr config view`: effective merged config with per-key source
      annotation; secrets rendered as `[redacted]`, `--show-secrets`
      reveals. Tests: token set in config → output shows `[redacted]`
      (fails without redaction wiring); with `--show-secrets` shows value;
      bare `rwr config` still prints help.
- [ ] 2. `rwr config edit`: `$EDITOR`/`$VISUAL`/platform fallback; absent
      file created via `CreateDefaultConfig` first; post-edit re-parse warns
      on invalid config. Tests: editor invoked with the config path (fake
      editor script); parse-error warning fires.
- [ ] 3. `rwr config create` subcommand; `--create`/`-c` marked deprecated
      pointing at it. Tests: both forms create; deprecated message names the
      subcommand.
- [ ] 4. Shorthands: `-n` → `--dry-run`, `-l` → `--log-level`. Tests: `-n`
      sets dry-run, `-l debug` sets the level; assert no shorthand collision
      across root + subcommand flag sets (fails if a future flag claims one).
- [ ] 5. Help-text pass: `-i`/`-I` cross-reference, `--interactive=false`
      guidance; regenerate/update docs flag tables and `commands/config.md`.
