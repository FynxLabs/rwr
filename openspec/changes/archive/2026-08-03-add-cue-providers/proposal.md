# Change: CUE provider definitions

Scope: provider definitions only. CUE as a *blueprint* format is
`add-cue-blueprints`, a separate change.

## Why

The 25 provider TOMLs (~1,845 lines) are the most duplicated and least checked
code in the repository:

- **Duplication.** yay, paru, trizen, aura and pamac are near-identical: same
  pacman-family commands, same detection files, same clone-and-makepkg install
  shape. Every family-wide fix (the `/tmp` staging move, `--needed`) is five
  edits that must not drift.
- **Weak validation.** TOML decoding accepts any shape; the checks live in
  `internal/validate/providers.go` as hand-rolled Go that has already drifted
  from the processors once. A wrong field name is silent until runtime.
- **No shared vocabulary.** Contracts like "a provider MUST declare detection
  and commands" exist only in reviewers' heads.

CUE fixes all three natively: schemas are unification (a provider lacking a
required field fails to *export*, at build time), and family templates are
plain values a definition unifies with.

Full design in `design.md`. Key decisions:

1. CUE is build-time only; `cuelang.org/go` never ships in the binary.
   `providers/cue/*.cue` → `cue export` → committed JSON → `go:embed`.
   Filesystem overrides stay TOML (and gain JSON).
2. Schema mirrors `types.Provider` exactly; a strict round-trip test enforces it.
3. Family templates for proven clusters only: `#PacmanFamily`,
   `#ArchAURHelper`, `#DebianFamily`.
4. Contracts move into the schema (required fields, action enum, condition
   predicate names cross-checked against `repositoryPredicates`, no literal
   `/tmp/` paths).
5. CI runs `cue vet` + export-freshness check; committed JSON keeps `go build`
   working without a CUE toolchain.

Resolved open questions: overrides accept TOML **and** JSON; `cue` pinned in
mise from PR 1; source lives at `providers/cue/` (repo root - it is source,
not an embedded asset).

## What Changes

- Provider definitions move from 25 TOML files to CUE sources at
  `providers/cue/` with a schema mirroring `types.Provider`; `cue export`
  produces the committed JSON the binary embeds.
- Family templates (`#PacmanFamily`, `#ArchAURHelper`, `#DebianFamily`)
  deduplicate proven clusters.
- Provider contracts (required fields, action enum, condition predicate
  names, no literal `/tmp/` staging) move into the CUE schema; Go validation
  remains only for filesystem overrides, which accept TOML and JSON.
- CI gains `cue vet` and an export-freshness gate.

## Breakage

Nothing breaks for existing blueprints or filesystem provider overrides. The
exported JSON must decode into the same `types.Provider` values the TOMLs
produce today (round-trip test is the gate).

## Impact

- Affected specs: `provider-detection`, `blueprint-validation`, `distribution`.
- Deleted: the 25 provider TOMLs, most of `validate/providers.go`, the
  hardcoded action-list assertions in `embedded_test.go`.
- Expected corpus: 1,845 → ~1,100 lines, plus ~150 schema / ~60 CI / ~80 tooling.
