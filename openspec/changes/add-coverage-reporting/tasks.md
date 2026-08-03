# Tasks

- [x] 1. Measure current per-package coverage; record baseline in the change.
- [x] 2. Add `test:coverage:html` / `test:coverage:summary` mise tasks.
- [x] 3. Gate script comparing total coverage to threshold; threshold set from
      the baseline.
- [x] 4. Wire the gate into the ubuntu CI test job; verify it fails on a
      simulated coverage drop (delete a test locally) and passes on master.
- [x] 5. README badge.
