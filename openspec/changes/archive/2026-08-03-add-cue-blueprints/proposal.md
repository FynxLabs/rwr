# Change: CUE as a supported blueprint format

Depends on: `add-format-registry`. Related: `add-cue-providers` (independent -
that change is build-time only; this one is runtime).

## Why

Blueprints today are YAML, JSON, or TOML - all shape-unchecked at authoring
time; errors surface at run time on the target machine. CUE gives blueprint
authors types, constraints, and composition (shared fragments unified across
machines), and rwr already publishes strict schemas per blueprint type that
CUE can encode.

## What Changes

- `.cue` joins the known extensions in the format registry: init files,
  bootstrap files, blueprints, imports, and (once it exists) the manifest.
- **Runtime dependency on `cuelang.org/go`** (decision): the user's `.cue` is
  evaluated in-process and exported to concrete JSON, which feeds the existing
  strict-decode path (`DecodeBlueprintInto`). CUE evaluation errors are
  surfaced as diagnostics with file/line.
  - Rejected alternative: shelling out to a `cue` binary. rwr's job is
    bootstrapping machines that don't have tools yet; requiring `cue` on the
    target defeats it. Cost of the dep is binary size (~+10–15 MB), accepted.
- Init-file path: CUE is evaluated → exported to JSON → handed to the existing
  viper decode (mirroring today's TOML→YAML pre-conversion).
- `rwr validate` runs CUE evaluation as part of stage 1 resolve; a CUE value
  that doesn't unify is a validation error, same surface as schema errors.
- `examples/` gains a fourth format column for every blueprint type per the
  compatibility contract; validated in CI like the other three.
- Optional (follow-up-friendly): ship rwr's blueprint schemas as importable
  CUE definitions so authors can write `packages: #Packages & {...}`.

## Breakage

Nothing breaks. Existing formats are untouched; `.cue` is additive. A tree may
mix `.cue` with other formats (per-file format derivation from
`add-format-registry`).

## Impact

- Affected specs: `blueprint-processing`, `initialization`,
  `blueprint-validation`, `blueprint-schema-versioning`.
- CUE blueprints are still untrusted input: evaluation must run with no
  filesystem/network access outside the blueprint tree (CUE `@embed`/module
  resolution disabled or rooted at the tree).
