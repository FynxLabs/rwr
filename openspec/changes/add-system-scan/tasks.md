# Tasks

- [ ] 1. `list_explicit` in the provider CUE schema + the definitions that
      have one (pacman family, apt, dnf, brew, cargo, npm-family, zypper,
      apk, chocolatey/scoop/winget where supported); export pipeline and
      round-trip tests updated. Test: a provider without the verb still
      exports and loads.
- [ ] 2. `internal/scan` package scanner: per detected provider, parse
      explicit output (fixtures per provider, like the status querier);
      fall back to `list` marked unfiltered. Test: fixtures per provider;
      a scan never executes a non-list verb (trap-binary test, same
      pattern as status).
- [ ] 3. Config scanner: known-dotfile set + `~/.config` top level, shipped
      noise exclusion list, results carry path + kind. Test: fixture home
      dir; excluded entries absent; unknown entries present.
- [ ] 4. Service scanner: enabled units minus vendor presets on linux;
      `unknown` off-platform. Test: parse fixtures.
- [ ] 5. Git scanner: clones under configured roots with remote URL +
      path. Test: fixture tree.
- [ ] 6. Emission: scan results → blueprint block in any registry format,
      reusing the convert encoders. Test: golden output per format,
      round-trips through strict decode.
