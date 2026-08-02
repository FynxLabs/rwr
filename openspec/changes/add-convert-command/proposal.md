# Change: `rwr convert` — format conversion and tree migration

Depends on: `add-format-registry` (decode/encode dispatch),
`unify-blueprint-semantics` (defines the "current state" trees migrate to).
Status: stub — requested alongside the decision to drop init-file inline
sections; needs fleshing out before implementation.

## Why

Two recurring needs with no tool today:

- **Format conversion**: a tree written in YAML has no path to TOML/JSON/CUE
  short of hand-rewriting every file, even though all four formats decode to
  identical values.
- **Migration to current state**: schema evolution (dropped init-file inline
  sections, future schema_version bumps) leaves working-but-deprecated trees
  behind, with only release notes to guide the move.

## What Changes (sketch)

- `rwr convert --to <format> [path]`: re-encode every blueprint, init, and
  bootstrap file in a tree to the target format, preserving comments where the
  target supports them and template placeholders byte-identical.
- `rwr convert --migrate [path]`: rewrite deprecated constructs to their
  current equivalents — starting with init-file inline resource sections moved
  into blueprint files under the tree.
- Dry-run by default with a diff; `--write` applies.

## Breakage

None — a new command; it only writes with `--write`.

## Impact

- Affected specs: `cli` (new command), possibly `initialization`.
