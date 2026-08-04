# Change: Replace cmd package globals with a config struct

(Was issue #104.)

## Why

`cmd/root.go` holds 17 package-level mutable variables (`ghApiToken`, `debug`,
`profiles`, `initConfig`, `osInfo`, …). They make unit testing the cmd package
nearly impossible (no isolated state, no reset between tests), risk races in
parallel tests, and hide ownership/lifecycle.

## What Changes

- An `AppConfig` struct owning all current globals (auth, execution flags,
  paths, active `InitConfig`/`OSInfo`/profiles), constructed in
  `PersistentPreRunE` and passed to `initializeSystemInfo` and the command
  implementations.
- Cobra flags bind to struct fields; no package-level mutable state remains in
  `cmd` beyond the cobra command tree itself.

## Breakage

Nothing breaks. Flags, config keys, and behavior are unchanged; this is an
internal restructuring.

## Impact

- Affected specs: `cli` (no requirement changes - internal only).
- Makes cmd-level tests with isolated config instances possible.
