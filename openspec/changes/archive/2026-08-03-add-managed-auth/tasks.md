# Tasks

- [x] 1. Types: `credentials:` init-file section (strict decode; unknown keys
      error), credential registry replacing the fixed `SecretConfigKeys` list
      while keeping the two built-ins implicitly declared. Test: a tree
      declaring `cachix_token` gets it withheld/redacted exactly like
      `gh_api_token`; fails without the registry.
- [x] 2. Source resolvers: `env:<VAR>`, `keyring`, `prompt` behind one
      interface; ordered first-hit-wins resolution. Table tests per source and
      per precedence order, keyring faked.
- [x] 3. Resolution phase in `ProcessInitialization` (after
      `SetExposedCredentials`, before `setUserDefinedAndEnvVariables`).
      Tests: unresolvable credential errors up front naming sources tried;
      non-TTY skips `prompt` and the error says so; out-of-scope credentials
      for the selected processors are not resolved.
- [x] 4. Keyring store (zalando/go-keyring or equivalent): read source +
      explicit save; unavailable backend degrades silently on read, errors on
      save. Interface-level tests with a fake; no CI dependency on a real
      keyring.
- [x] 5. Redaction + export: resolved values registered for redaction; env
      export as `RWR_CRED_<NAME>` gated by `exposeCredentials` and `scope`;
      template scope under `.Credentials.<name>` behind the same gate. Tests:
      value absent from spawned env and rendered templates without opt-in,
      present with; debug logs show `[redacted]` (fails without redaction
      registration); value never in argv.
- [x] 6. Fold the GitHub token in: `--gh-auth` / `--gh-api-key` populate the
      `gh_api_token` credential; device-flow save prefers keyring, config-file
      fallback warns with the path. Tests: existing config-file token still
      read; keyring-available path writes no plaintext.
      *Deviation:* the built-ins' implicit sources omit `prompt` at init time -
      prompting for the GitHub token stays at its point of use (`ssh_keys`),
      preserving the existing behavior of never prompting a run that does not
      need the token. Built-ins are optional: unresolved is not an error.
- [x] 7. Prompt UX: masked huh input with `description`, post-prompt
      save-to-keyring offer, honored `--interactive=false`.
- [x] 8. Docs (`configuration.md`, new credentials page) + `examples/` init
      files covering `credentials:` in all formats; validate covers the
      section.
      *Deviation:* the credentials page already existed (`docs/credentials.md`)
      and was extended rather than created; the init-file schema doc is
      `docs/init-file.md` (there is no init `configuration.md`), and
      `docs/variables.md` documents the `.Credentials` namespace and
      `RWR_CRED_*` export. Examples: `examples/macos/{yaml,json,toml,cue}`.
      Validation of the section happens at load (strict decode in
      `Initialize`) and in the examples strict-schema/parity tests; `rwr
      validate` does not parse init files today, so no change there.
