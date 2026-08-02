# Tasks

- [ ] 1. Manifest types + strict decode via the format registry; reject `init`
      paths escaping the repo root (test with `../` fails without the check).
- [ ] 2. Init resolution: detect manifest at root when no init file present;
      test repo-root and git-URL forms.
- [ ] 3. Matcher filtering against `types.OSInfo` (os/distro/family/arch);
      table tests: zero, one, many matches.
- [ ] 4. `--config-name` flag; non-TTY with multiple matches errors and lists
      candidates (test fails without).
- [ ] 5. TUI selection frame (post `add-tui`): entry list with matched entries
      first, renders before resolve stage 1.
- [ ] 6. Example manifest under `examples/`; validate covers manifest files.
