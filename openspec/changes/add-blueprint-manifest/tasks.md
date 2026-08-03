# Tasks

- [x] 1. Manifest types + strict decode via the format registry; reject `init`
      paths escaping the repo root (test with `../` fails without the check).
- [x] 2. Init resolution: detect manifest at root when no init file present;
      test repo-root and git-URL forms.
- [x] 3. Matcher filtering against `types.OSInfo` (os/distro/family/arch);
      table tests: zero, one, many matches.
- [x] 4. `--config-name` flag; non-TTY with multiple matches errors and lists
      candidates (test fails without).
- [x] 5. TUI selection frame (post `add-tui`): entry list with matched entries
      first, renders before resolve stage 1.
- [x] 6. Example manifest under `examples/`; validate covers manifest files.
