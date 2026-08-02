# Change: Centralize blueprint format handling (format registry)

## Why

Format *decoding* is centralized (`internal/helpers/blueprints.go` `unmarshalBlueprint`),
but format *derivation* — which extensions exist and which decoder a file gets — is
hardcoded in ~15 places. Worse, the codebase holds two contradictory models:

- `internal/processors/blueprints.go:187,221` and `internal/processors/profiles.go:53`
  assume the whole tree is one format via `initConfig.Init.Format`.
- `internal/processors/all.go:148` and the validate package derive format per file
  from its extension. `all.go:148` (`filepath.Ext(file)[1:]`) panics on an
  extensionless file.

Every hardcoded site is a place a new format (CUE) must be added by hand, and the
two models disagree today for a tree that mixes formats. This change is a
prerequisite for `add-cue-blueprints` and `add-blueprint-manifest`.

Known hardcoded/derived sites: `cmd/root.go:114-119`, `cmd/root.go:130`,
`cmd/validate.go:74`, `internal/types/constants.go:21-29`,
`internal/processors/all.go:22,148`, `internal/processors/initialize.go:100,186`,
`internal/processors/bootstrap.go:46,51`, `internal/processors/blueprints.go:187,221`,
`internal/processors/profiles.go:53,65,68`, `internal/helpers/resolve_imports.go:70`,
`internal/validate/blueprints.go:48,64,92,99,122,212,231`,
`internal/validate/helpers.go:113,117,122`.

Also fixed here: `--init-file` is bound to viper key `rwr.init-file`
(`cmd/root.go:206`) but read from `repository.init-file` (`cmd/root.go:105`), so
the config-file form of the setting and the flag disagree.

## What Changes

- One registry in `internal/helpers` (or `internal/types`) owning: known
  extensions, extension → format resolution, candidate init/bootstrap filename
  generation, and decoder dispatch. All sites above call it.
- Format is derived **per file** everywhere. `Init.Format` becomes a fallback/
  default for extensionless cases only (or is deprecated); the tree-uniform
  assumption in `blueprints.go`/`profiles.go` is removed.
- Extensionless and unknown-extension files produce a diagnostic, not a panic.
- The init-file path also flows through the registry for discovery (viper decode
  can remain for now; `add-cue-blueprints` revisits it).
- Fix the `rwr.init-file` / `repository.init-file` viper binding mismatch.

## Breakage

Nothing breaks for existing blueprints. A tree that mixes formats — previously
undefined behavior (half the code honored it, half didn't) — becomes supported.
`examples/` must still pass unchanged.

## Impact

- Affected specs: `blueprint-processing`, `initialization`, `blueprint-validation`, `cli`.
- Unblocks: `add-cue-blueprints`, `add-blueprint-manifest`.
