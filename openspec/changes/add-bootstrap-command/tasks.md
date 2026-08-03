# Tasks

- [x] 1. Add `bootstrap` to the `runProcessors` table with a dedicated
      dispatch to `ProcessBootstrap` (the generic `processors.All` path
      doesn't run bootstrap). Test: `rwr run bootstrap` and root shorthand
      `rwr bootstrap` both resolve (fails without the entry).
- [x] 2. Explicit invocation bypasses the run-once marker: standalone
      bootstrap runs even when `<configdir>/bootstrap` exists, and refreshes
      the marker on success. Test: marker present, standalone run still
      processes; `rwr all` without `--force-bootstrap` still skips (fails if
      gating leaks).
- [x] 3. Missing bootstrap file errors, naming the candidate filenames
      searched. Test: tree without a bootstrap file → non-zero exit and the
      candidates in the message.
- [x] 4. `--dry-run` respected end to end (no marker write, no mutations);
      test via the central dry-run path.
- [x] 5. Docs: commands page for `rwr bootstrap`; note the relationship to
      `--force-bootstrap` on `all`.
