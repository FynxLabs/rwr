## Context

See proposal.md. The approved detailed design is `.local/credentials-processor-plan.md`. The old execution loop resolves all scoped declarations before profile selection and exports values process-wide.

## Goals / Non-Goals

Use native Go orchestration and explicit sessions; official CLI argv calls remain supported. Bitwarden ships first. Future vendor adapters, Omarchy implementation, and live personal vault operations are outside this change.

## Decisions

- Route `credential_setup` to `credentials`, preserving init `credentials` references. Trust connection endpoints and attachment bindings only from init.
- Compute exclusions before preparation; named commands override init exclusions, CLI exclusions win. Explicit setup is the default; ordered setup is opt-in.
- Resolve dependencies after profiles; unavailable consumers skip by default, strict policy records failure. Resolution never prompts or installs.
- Keep secret templates symbolic through structural decoding, resolve at the consuming resource, and pass command secrets only in child environments or protected input.
- Provider registry uses per-run sessions and optional attachment capabilities. Native tasks validate GPG artifacts in isolated keyrings before real changes. Backup requires a selected write profile and verifies new attachments before deleting old ones.
- Reuse the existing verified Bitwarden installer. Setup owns interaction; runtime access is prompt-free. Persistence is explicit, account-namespaced keyring storage; session tokens stay in memory.

## Risks / Trade-offs

- Existing scripts can call vault CLIs themselves → migrate the personal GPG scripts and document declared dependencies for opaque scripts.
- Vendor CLI errors may contain secrets → protected output with categorical diagnostics and no raw stderr.
- Existing global template registry → resource-lifetime values, cleanup, and leakage tests while preserving public compatibility.
- No live account verification → fake adapters and disposable GPG key material exercise lifecycle and task failures.

## Migration Plan

Ship schema, CLI, native processor and regression coverage together. Document changed acquisition behavior. Migrate personal blueprints on a separate branch after core validation; retain unrelated machine customizations. Reverting branches restores old declarations; no production vault changes are performed by development tests.
