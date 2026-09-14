## Why

Credential lookup currently starts provider onboarding before the selected consumers run. Missing or locked Bitwarden can interrupt unrelated setup, including scripts whose credential-dependent profiles were not selected.

## What Changes

- Add an explicit, repeatable credentials processor with native Bitwarden setup and credential tasks.
- Add generic CLI and init processor exclusions, applied before acquisition.
- **BREAKING**: ordinary runs use noninteractive, demand-based lookup; they never install, log in, unlock, or prompt for declared credentials.
- Add resource dependency policy, scoped secret handling, native GPG restore/backup, and trusted connection/attachment declarations.
- Migrate documentation, examples, and the personal blueprint workflow; verify with isolated providers and test key material.

## Capabilities

### New Capabilities
- `credential-setup`: Explicit provider lifecycle, native tasks, selection policy, and safe session handling.

### Modified Capabilities
- `credential-handling`: Resolve only selected consumers' dependencies without implicit onboarding.
- `cli`: Generic processor exclusions and a credentials command.

## Impact

CLI, init/schema registries, blueprint routing/planning, processors, credential adapters, reporting, tests, and personal blueprints. Bitwarden is the initial adapter; other vendors remain extension points. No live vault or desktop provisioning is part of development validation.
