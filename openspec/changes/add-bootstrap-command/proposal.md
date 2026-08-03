# Change: `rwr bootstrap` as a standalone command

## Why

Bootstrap is the one processor you cannot run by itself. It runs first inside
`rwr all` when a bootstrap file exists (`internal/processors/all.go:123-129`),
gated by a run-once marker: `ProcessBootstrap` skips unless
`--force-bootstrap` is set or the marker is absent
(`internal/processors/bootstrap.go:18-21`; the marker is the empty file
`<configdir>/bootstrap`, `internal/helpers/bootstrap.go:13-32`). It is
deliberately absent from the `runProcessors` table (`cmd/run.go:112-123`), so
neither `rwr run bootstrap` nor the root shorthand `rwr bootstrap` exists.

The result: re-running just the bootstrap after editing `bootstrap.yaml`
requires `rwr all --force-bootstrap` — which also re-runs every other
processor. That is both slow and surprising ("I asked to redo bootstrap, why
is it reinstalling packages?").

## What Changes

- `rwr run bootstrap` joins the processor table, and therefore the root
  shorthand `rwr bootstrap` works too — same mechanism as every other
  processor, no special-casing in `cmd/`.
- Semantics of an explicit invocation: asking for bootstrap by name implies
  wanting it to run, so the standalone command ignores the run-once marker
  (equivalent to `--force-bootstrap`) — the marker exists to keep `all`
  idempotent, not to refuse an explicit request. It still writes/refreshes the
  marker on success and still honors `--dry-run`.
- No bootstrap file in the tree → clear error naming the candidate filenames
  looked for (reusing `findBootstrapFile`'s candidates,
  `internal/processors/all.go:20-31`), rather than a silent no-op.
- `rwr all --force-bootstrap` keeps working unchanged.

## Breakage

Nothing breaks. New subcommand and shorthand; `all`'s gating, the marker file,
and `--force-bootstrap` are untouched. The only new name claimed at the root
is `bootstrap`, which was previously an unknown-command error.

## What this is not

Not a redesign of the run-once mechanism (the marker stays an empty file in
the config dir; a richer record is `add-uninstall-status`'s territory), and
not a change to bootstrap's contents or ordering inside `all`.

## Impact

- Affected specs: `cli`.
- Affected code: `cmd/run.go` (one `runProcessorSpec` entry + a dispatch that
  calls `ProcessBootstrap` instead of `processors.All`),
  `internal/processors/bootstrap.go` (accept an explicit-run flag or the
  caller passes force), docs `commands/` page.
