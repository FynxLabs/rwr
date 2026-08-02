# Tasks

- [ ] 1. Measure current per-package coverage; record baseline in the change.
- [ ] 2. Add `test:coverage:html` / `test:coverage:summary` mise tasks.
- [ ] 3. Gate script comparing total coverage to threshold; threshold set from
      the baseline.
- [ ] 4. Wire the gate into the ubuntu CI test job; verify it fails on a
      simulated coverage drop (delete a test locally) and passes on master.
- [ ] 5. README badge.
