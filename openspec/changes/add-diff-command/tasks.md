# Tasks

- [x] 1. `internal/diff`: scan-vs-tree comparison per category — additions
      (on machine, not in tree) and removals (in tree, gone from machine),
      package identity matched provider-aware. Test: synthetic tree +
      fixture scan → expected delta; blueprint-installed packages (journal
      entries) never report as hand-added.
- [x] 2. `rwr diff` list output grouped by category/provider; exit 0 always
      (reporting, not gating — `rwr status` owns the exit code). Test:
      golden output.
- [x] 3. `--format cue|yaml|json|toml` paste-ready blocks via the scan
      emission layer; `--packages`/`--configs`/`--services`/`--git` scoping.
      Test: emitted block strict-decodes; scoped runs touch only their
      scanners.
- [x] 4. `--into <tree>`: destination discovery (machine tree file for the
      category, Common files reachable via its imports, skip), huh form per
      change group, edit written in the target file's own format. Tests:
      destination discovery on a fixture tree with Common imports;
      non-interactive `--into` without a TTY errors naming `--format` as
      the alternative; written file strict-decodes and `rwr validate`
      passes.
- [x] 5. Docs: `docs/cli/diff.md`, cross-link from state.md's status
      section.
