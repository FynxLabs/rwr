# Tasks

- [x] 1. Add format registry: extensions set, `FormatForPath(path) (Format, error)`,
      `CandidateFilenames(base) []string`, decoder dispatch. Unit tests, including
      extensionless and unknown extensions (fails today via `all.go:148` panic).
- [x] 2. Replace the hardcoded extension lists in `cmd/`, `internal/processors/`,
      `internal/helpers/`, `internal/validate/` with registry calls.
- [x] 3. Remove tree-uniform `Init.Format` assumption in
      `processors/blueprints.go` and `processors/profiles.go`; add a test with a
      mixed-format tree (fails without the change).
- [x] 4. Fix `rwr.init-file` vs `repository.init-file` binding; test that
      `--init-file` and the config key resolve identically (fails today).
- [x] 5. `mise run ci` green; `examples/` unchanged and passing.
