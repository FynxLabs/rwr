# Design: CUE provider definitions

Status: proposed
Scope: provider definitions only. CUE as a *blueprint* language is a follow-up
with its own design once this lands (`add-cue-blueprints`).

## Why

The 25 provider TOMLs (~1,800 lines) are the most duplicated and least checked
code in the repository:

- **Duplication.** yay, paru, trizen, aura and pamac are near-identical: same
  pacman-family commands, same detection files, same clone-and-makepkg install
  shape, differing in a name and a URL. Every family-wide fix (the `/tmp`
  staging move, the `--needed` flag) is five edits that must not drift.
- **Weak validation.** TOML decoding accepts any shape; the checks live in
  `internal/validate/providers.go` as hand-rolled Go that has already drifted
  from the processors once (the `condition`/`path` fields, the fictional
  actions). A wrong field name is silent until runtime.
- **No shared vocabulary.** Nothing says "a provider MUST declare detection
  and commands", or "a repository add step that names a condition must use a
  derivable predicate". Those contracts exist only in reviewers' heads.

CUE fixes all three natively: schemas are unification (a provider that lacks a
required field fails to *export*, at build time), and family templates are
plain values a definition unifies with (`yay: #ArchAURHelper & {name: "yay"}`).

## Decisions

1. **CUE is a build-time tool, not a runtime dependency.** `cuelang.org/go`
   never ships in the binary. The pipeline is:

   ```
   providers/cue/*.cue  --cue export-->  internal/system/definitions/providers/*.json  --go:embed-->  binary
   ```

   The exported JSON decodes into the existing `types.Provider` structs; the
   loading code changes from a TOML decoder to a JSON decoder and nothing
   downstream notices. Filesystem provider *overrides* stay TOML (or gain
   JSON) - operators should not need CUE installed to override a provider.

2. **Schema mirrors `types.Provider` exactly, and a test enforces it.** A Go
   test round-trips every exported provider through strict decoding into
   `types.Provider` with no unknown keys, so the CUE schema and the Go structs
   cannot drift apart silently - the same guarantee `embedded_test.go` gives
   today, but at the schema level.

3. **Family templates for the proven clusters only.**
   - `#PacmanFamily`: commands + detection files + repository paths shared by
     pacman and the AUR helpers.
   - `#ArchAURHelper`: `#PacmanFamily` + the clone-and-makepkg install steps,
     parameterized by AUR package name (collapses 5 files to ~5 lines each).
   - `#DebianFamily`: apt/apt-get share detection and repository shape.

   Everything else stays a standalone definition - inventing hierarchies for
   single members obscures more than it saves.

4. **Contracts move into the schema.** What `validate/providers.go` checks by
   hand becomes unification:
   - required: `name`, `detection.binary`, `commands.install`
   - `elevated` defaults to false, must be explicit `true` for system managers
   - step actions constrained to the enum the processors implement
   - a `condition` may reference only the predicate names rwr derives
     (kept as a CUE `or` list that a Go test asserts equals the keys of
     `repositoryPredicates` - one source of truth, checked both ways)
   - install/remove steps may not contain literal `/tmp/` paths
     (the `{{ .TempDir }}` rule, today a Go test, becomes a schema regexp)

5. **CI validates and checks freshness; the export is committed.** A
   `cue-providers` job runs `cue vet` + `cue export` and fails if the
   committed JSON differs from the fresh export (same pattern as gofmt).
   Committed output keeps `go build` working with no CUE toolchain installed -
   contributors touching providers need CUE (via mise), everyone else doesn't.

## What gets deleted

- The 25 TOML files under `internal/system/definitions/providers/`.
- Most of `internal/validate/providers.go` (schema does it); what stays is the
  filesystem-override validation, which still sees TOML.
- The hardcoded action list assertions in `embedded_test.go` (schema enum).

## Expected size

Corpus 1,845 → ~1,100 lines (5 AUR files collapse to ~25 lines total, shared
commands/paths lift into families). Plus ~150 lines of schema, ~60 of CI, ~80
of export-freshness tooling.

## Sequencing (3 PRs)

1. **Schema + pipeline**: `providers/cue/schema.cue`, export tooling
   (`mise run providers:export`), CI job, loader reads committed JSON -
   with the TOMLs still the source, converted mechanically 1:1 (no family
   templates yet). Proves the pipeline with zero semantic change; the
   round-trip test is the gate.
2. **Families**: introduce `#PacmanFamily` / `#ArchAURHelper` /
   `#DebianFamily`, collapse the members onto them. Exported JSON must be
   byte-identical to PR 1's - the diff *is* the review.
3. **Contracts**: move the validate/providers.go checks into the schema,
   delete the superseded Go, add the predicate-name cross-check test.

## Resolved questions

- Filesystem overrides: accept both `.toml` and `.json`; JSON is what the
  export produces, so power users can copy one out and edit it.
- `mise install` pins `cue` from PR 1 (CI needs it from day one).
- Source lives at `providers/cue/` (repo root): it is source, not an embedded
  asset.
