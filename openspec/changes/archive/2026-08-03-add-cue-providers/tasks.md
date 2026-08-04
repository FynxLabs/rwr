# Tasks

Three PRs, each independently green.

- [x] 1. **Schema + pipeline**: `providers/cue/schema.cue`, mechanical 1:1
      conversion of the 25 TOMLs, `mise run providers:export`, pinned `cue` in
      mise, CI job (`cue vet` + freshness diff), loader reads committed JSON.
      Gate: strict round-trip test - every exported provider decodes into
      `types.Provider` with no unknown keys and equals the TOML-derived value
      (fails on any semantic drift).
- [x] 2. **Families**: `#PacmanFamily`, `#ArchAURHelper`, `#DebianFamily`;
      collapse members. Gate: exported JSON byte-identical to PR 1's.
- [x] 3. **Contracts**: move `validate/providers.go` checks into the schema;
      delete superseded Go; predicate-name cross-check test (CUE `or` list ==
      keys of `repositoryPredicates`, asserted from Go - fails if either side
      drifts). Filesystem overrides accept `.toml` and `.json`; override
      validation stays in Go.
