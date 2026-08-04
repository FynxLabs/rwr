# Change: System scan — the shared harvest layer for diff and capture

Scope: the scanning foundation only. The two consumers — `rwr diff`
(add-diff-command) and `rwr capture` (add-capture-command) — are separate
changes that depend on this one.

## Why

Two upcoming commands need the same answer to the same question: "what is
actually on this machine, filtered to what a human chose to put there?"

- `rwr diff` compares that answer against the blueprint tree and offers the
  delta as blueprint updates.
- `rwr capture` takes that answer on a machine with no tree at all and
  generates one.

Today rwr can only see what a blueprint declared (`internal/processors/plan.go`)
or what a run recorded (`internal/state`). The status querier
(`internal/status/query.go`) reads provider `list` output, but that is the
full installed set — on Arch that is every dependency ever pulled in, ~1200
packages of noise around the ~200 the operator actually asked for. There is
no notion of "explicitly installed", no config scanner, and no service
scanner beyond the single-unit `is-enabled` probe.

## What Changes

- A `list_explicit` command joins the provider CUE schema and definitions:
  the manager's explicitly-installed query (`pacman -Qe`, `apt-mark
  showmanual`, `brew leaves`, `dnf repoquery --userinstalled`,
  `cargo install --list`, `npm ls -g --depth=0`, …). Providers without one
  fall back to `list` and are marked unfiltered in scan results.
- `internal/scan`: read-only scanners returning typed results —
  - packages: per detected provider, explicit set preferred;
  - configs: the known-dotfile set (`.bashrc`, `.gitconfig`, …) plus
    top-level `~/.config` entries, minus a shipped known-noise exclusion
    list (caches, state dirs, session junk);
  - services: enabled units the operator enabled (linux: `systemctl
    list-unit-files --state=enabled`, minus vendor presets where
    determinable), per-platform, honest `unknown` elsewhere;
  - git checkouts: clones under configured roots (default `~/git`).
- Scanners never mutate and never elevate — same contract as the status
  querier, enforced the same way.
- Blueprint emission helpers: a scan result renders as a blueprint block in
  any registry format (cue/yaml/json/toml), reusing the format-registry
  encoders the convert command established.

## Breakage

None. Additive schema field (providers without `list_explicit` keep working),
new internal package, no CLI surface of its own.

## Impact

- Affected specs: `system-scan` (new capability), `provider-detection`
  (schema field).
- Affected code: `providers/cue/schema.cue` + definitions, new
  `internal/scan`, format emission shared with `internal/convert`.
- Consumers: add-diff-command, add-capture-command.
