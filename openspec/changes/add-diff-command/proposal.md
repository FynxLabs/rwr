# Change: rwr diff — machine drift as blueprint updates

Depends on: add-system-scan (the harvest layer).

## Why

A provisioned machine drifts: packages installed by hand, configs adjusted,
services enabled. `rwr status` (state-tracking spec) answers "does the
machine match the tree" — but its unit of report is drift, and its consumer
is an exit code. The operator's actual next step is "fold the drift I want
to keep back into the blueprints", and today that is a manual diff of
`pacman -Qe` against yaml by eyeball.

The journal (`internal/state`) makes this tractable: rwr knows what *it*
applied, so hand-installed work is separable from blueprint-installed work
instead of guessed at.

## What Changes

- `rwr diff`: compares the scan (add-system-scan) against the resolved
  blueprint tree and reports what is on the machine but not in the tree,
  and what the tree declares that is gone from the machine.
- Output modes:
  - default: a readable list grouped by category and provider;
  - `--format cue|yaml|json|toml`: paste-ready blueprint blocks of the
    additions/removals, rendered by the scan emission layer;
  - `--packages` (and siblings) to scope to one category.
- `--into <tree>`: interactive routing. For each change group a huh form
  offers the destination candidates discovered from the tree itself — the
  matching machine tree's blueprint file, any Common file the tree imports
  for that category, or skip — then writes the edit in the target file's
  own format. rwr cannot know whether a new package is an Archcraft thing
  or a Common thing; that is the operator's call, made once per group
  instead of once per hand-edit.
- Never applies anything: diff writes blueprint files when told to, and the
  system only when the operator later runs `rwr all`.

## Breakage

None. New command; no existing surface changes.

## Impact

- Affected specs: `cli` (new command), reads `system-scan` and
  `state-tracking`.
- Affected code: `cmd/diff.go` (wiring), `internal/diff` (comparison +
  routing), huh form (dependency already present).
