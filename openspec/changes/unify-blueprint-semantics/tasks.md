# Tasks

- [x] 1. `key_sha256` on `types.Repository`, threaded into step data
      (`KeySha256`) and the apt/dnf key-download steps' `sha256`. Warn on
      `key_url` without it. Test: pinned mismatch refuses (fails without);
      unpinned warns.
- [x] 2. `sha256` on `types.File`; `files.go` URL downloads verify it (the
      call currently hard-codes ""). Warn on URL source without it. Test:
      mismatch refuses and installs nothing.
- [x] 3. File-relative import resolution in the six root-relative processors
      (packages, git, ssh_keys, services, users, repositories). Test: the
      same relative import string resolves identically from two blueprint
      types (fails without).
- [x] 4. `names` wins over `name` in packages (matching files/fonts); validate
      warns when both are set. Tests for both.
- [x] 5. Remove init-file inline resource sections from `types.Init*`,
      validation, profile counting, and docs (`docs/init-file.md`). Strict
      decode makes a leftover key an actionable error. Test: init file with
      `packages:` fails naming the key.
- [x] 6. Content-based type detection for path-unrouted files in
      `GetBlueprintFileOrder` (single-type match dispatches; zero or multiple
      matches keep the loud warning). Un-dead-end
      `examples/alternative_layouts/` (CI-validated), fix their READMEs'
      `rwr run` instruction. Test: flattened tree executes its packages file
      (fails without).
- [x] 7. Validate template strictness: `missingkey` strict for
      User/System/Flags, lenient for UserDefined only
      (`ResolveTemplateForValidation`). Test: `{{ .User.hoem }}` is an error,
      `{{ .UserDefined.x }}` is not (fails without).
- [x] 8. Delete tautological tests encountered along the way; `mise run ci`
      green; examples pass.
