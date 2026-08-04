# Tasks

- [x] 1. `internal/capture`: selection model over scan results with
      per-category defaults (explicit packages on, known dotfiles on,
      services off, git checkouts on). Test: defaults per category.
- [x] 2. huh multi-page form: one page per category, counts in titles,
      select-all/none per page; `--all` skips the form taking defaults.
      Test: `--all` selection equals defaults; form model unit-tested
      without a TTY.
- [x] 3. Tree generation: init + per-category blueprint files in the chosen
      format; selected configs copied under `files/src/` with file entries
      targeting their origin via `{{ .User.home }}`. Test: golden tree from
      a fixture scan; secrets-shaped paths (`~/.ssh`, key material) are
      never auto-selected and warn when selected explicitly.
- [x] 4. `--manifest`: root manifest with the machine's detected OS/distro
      matchers. Test: generated manifest strict-decodes and matches the
      fixture OS.
- [x] 5. Self-check: capture runs `rwr validate` on the generated tree and
      fails loudly if it fails (test: corrupt an emission → capture exits
      non-zero).
- [x] 6. Docs: `docs/cli/capture.md` + a "start from an existing machine"
      section in quick-start.
