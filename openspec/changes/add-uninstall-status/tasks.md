# Tasks

- [x] 1. `internal/state`: versioned run-record types, incremental writer
      (`<configdir>/state/runs/`, user-only perms), append-only journal with
      `latest` pointer. Tests: crash-partial record is readable and marked
      unfinalized; dry-run writes nothing.
- [x] 2. Record emission from the per-item apply points in each processor
      (ride the existing failure-ledger seams). Tests per processor: applied
      item appears with correct identity (package name+provider, file
      dest+sha256, service name, repo name, font name, git target); failed
      item recorded as not-ok. (Emission rides the single progress seam all
      ten processors already use; the seam test covers provider+name and
      dest+sha256 identity plus failed-not-ok, and files/git enrich with
      dest+sha256 / checkout target.)
- [ ] 3. Query capability: packages via provider `list` (parse fixtures per
      provider), files via dest+sha256, services via platform query,
      repositories via source-file presence, fonts/git via path existence;
      scripts/configuration/users/ssh_keys return `unknown`. Tests: each
      query class, plus queries never execute a mutating command (fails
      without the read-only guard).
- [ ] 4. `rwr status`: desired via the plan machinery, actuals via queries,
      drift table (`in-sync/missing/modified/unknown/stale`), exit 1 on
      drift. Tests: golden output for a synthetic tree; no-record fallback
      omits the recorded column; command never elevates.
- [ ] 5. `rwr uninstall`: record-driven reverse-order removal — provider
      `remove` for packages, provider `remove` steps for repositories,
      hash-guarded deletes for files/dirs/git/fonts, disable/stop for
      services. Tests: modified file skipped and listed; already-absent
      package skipped; reverse ordering asserted.
- [ ] 6. Uninstall safety UX: up-front not-reversible list (scripts,
      configuration, users, ssh uploads), interactive confirm, `--yes`,
      `--dry-run` through `system.RunCommand`. Tests: refusal with no record;
      confirm declined → nothing runs; dry-run mutates nothing.
- [ ] 7. Partial-failure semantics: per-item continue + ledger + non-zero
      exit; failed entries stay unreversed and a re-run retries them (test
      fails without re-run coverage).
- [ ] 8. Docs: new commands page, state-file format doc; examples unaffected
      (no blueprint schema change) — verify `examples/` validation still
      green.
