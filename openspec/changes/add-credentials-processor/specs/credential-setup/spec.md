## Purpose

Enable repeatable credential provider setup after ordinary provisioning, with explicit native tasks and scoped access to secrets.

## ADDED Requirements

### Requirement: Explicit repeatable credential setup
RWR SHALL route `credential_setup` entries to a credentials processor, filter profiles before provider access, and permit installation and authentication only during selected setup. Ordinary all runs SHALL exclude setup unless configured as ordered. A credentials-only run MUST NOT execute unrelated bootstrap or package preparation.

#### Scenario: Deferred setup
- **WHEN** the operator runs `rwr run credentials --profile bitwarden` after provisioning
- **THEN** only matching credential entries run, regardless of init's all-run exclusions
- **AND** current readiness is discovered again without a run-once marker

### Requirement: Scoped provider capabilities
RWR SHALL authorize connections, scalar references, and attachments from trusted init declarations. Sessions MUST remain confined to the run. Sensitive output MUST NOT reach logs, argv, journals, or plaintext session state. Dry-run MUST NOT perform credential operations.

#### Scenario: Untrusted attachment
- **WHEN** a task references an undeclared attachment or another connection's binding
- **THEN** it fails before contacting a provider

### Requirement: Native credential tasks
RWR SHALL orchestrate keyring materialization and GPG restoration/backup natively. Restore MUST verify identity, complete key set, and passphrase in temporary storage before real import. Trust and signing configuration SHALL be explicit. Backup MUST require an explicitly selected write profile and verify replacement before deleting an old attachment.

#### Scenario: Wrong restored identity
- **WHEN** an attachment contains an unexpected key
- **THEN** the real keyring remains unchanged and the task fails

#### Scenario: Failed replacement
- **WHEN** backup upload or verification fails
- **THEN** the prior attachment remains available and the task fails
