# Change: Multi-configuration blueprint repos (root manifest + smart selection)

Depends on: `add-format-registry`, `add-tui` (selection renders as a TUI frame).

## Why

A single blueprint repo commonly carries several machines' worth of config —
e.g. TheFynx/rwr-blueprints has Arch/, Archcraft/, Common/, macOS/,
OpenMandriva/, PopOS/, Windows/ at root. Today rwr has no notion of this: the
operator must point `--init-file` at the right subdirectory by hand, and there
is no way to publish one repo URL that works on any of your machines.

## What Changes

- A `manifest.{yaml,yml,json,toml}` file at blueprint-repo root (`.cue` once
  `add-cue-blueprints` lands), listing named configurations. Each entry:
  - `name` — e.g. `arch-desktop`
  - `init` — path to that configuration's init file, relative to repo root
  - matchers: `os` (linux/darwin/windows), `distro`, `family` (arch/debian/…),
    optional `arch`
  - optional `default: true`
- Init resolution learns to accept a repo root (local dir or git URL): clone via
  the existing `GetBlueprints` machinery, find `manifest.*` at root, filter
  entries against `system.DetectOS()` (runs before init — ordering already works).
- Selection:
  - zero matches → error listing every entry and its matchers
  - exactly one match → use it, log which
  - multiple matches → TUI selection frame before resolve stage 1
    (stage 1 needs only `SetPaths` + `DetectOS`, so this composes with the TUI
    resolve pipeline); non-TTY runs require `--config-name`
- `--config-name <entry>` selects explicitly and always wins; scripts and CI
  never prompt.
- Shared content (`Common/`) needs no new mechanism: entries' init files reach
  it via existing relative imports.

## Breakage

Nothing breaks. Repos without a manifest behave exactly as today. The manifest
path only activates when `--init-file` points at a location whose root has a
`manifest.*` and no init file.

## Impact

- Affected specs: `initialization`, `cli`.
- Blueprint manifests are untrusted input like blueprints: `init` paths must
  resolve inside the repo (no `..` escape), matchers are data only.
