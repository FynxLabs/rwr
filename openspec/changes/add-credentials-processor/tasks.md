## 1. Selection and acquisition

- [x] 1.1 Add normalized CLI/init exclusions and setup policy; test precedence, conflicts, empty selection, and bootstrap isolation.
- [x] 1.2 Remove eager acquisition from all initialization paths; test scripts with absent/locked/excluded providers and filtered consumers.
- [x] 1.3 Add dependency-based lookup, late rendering, and scoped child injection; test skip/fail, exposure, dry-run, and cleanup.

## 2. Native credentials processor

- [x] 2.1 Register strict credential setup/connection schemas across formats, routing, profiles, validation, planning, CLI and reports; verify round trips and malformed references.
- [x] 2.2 Add provider registry and scoped Bitwarden setup/session adapter; test installation/authentication/skip/cancellation, account isolation, no hidden prompts, and secret-safe output.
- [x] 2.3 Add explicit keyring materialization and native GPG restore; test identity/passphrase validation, already-present behavior, permissions, cleanup, and explicit signing/trust.
- [x] 2.4 Add selected GPG backup with attachment verification before replacement; test write selection, failed upload, and old-backup retention.

## 3. Integration and migration

- [x] 3.1 Update docs, examples, status/uninstall behavior and compatibility guidance; validate examples and secret-free reporting.
- [x] 3.2 Migrate personal blueprints on a separate branch to native setup/tasks; validate with the new binary without live vault operations.
- [x] 3.3 Run full tests, race checks, lint, security and relevant platform builds; review diff and record verification results.
