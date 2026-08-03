# Change: `rwr convert` — format conversion and tree migration

Depends on: `add-format-registry` (decode/encode dispatch),
`unify-blueprint-semantics` (defines the "current state" trees migrate to).
Status: fleshed out (task 1). Decisions:

- **Comments are not preserved.** Cross-format conversion goes through a
  decode/encode cycle and no format's comment model maps onto another's;
  promising partial preservation invites silent loss. The command WARNS when
  a source file carries comments so the operator knows to port them by hand.
- **Template placeholders round-trip as strings.** `{{ .User.home }}` inside
  a quoted scalar decodes and re-encodes byte-preserved. A file whose
  templates make it unparseable raw (unquoted `{{` at value start in YAML)
  cannot be converted and is reported per file, not silently mangled.
- **CUE export style is JSON-form CUE** — valid CUE, lossless, and
  mechanical. Idiomatic CUE (constraints, unification) is authoring work a
  converter should not guess at.
- **Migration rules are a registry** (`migrateRules`), one entry per
  deprecation, each: detect(tree) → describe → apply. First rule:
  init-file inline resource sections move into blueprint files
  (`packages:` → `packages/from-init.<fmt>`, …).

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
