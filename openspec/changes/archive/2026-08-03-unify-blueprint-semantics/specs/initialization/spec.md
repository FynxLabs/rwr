# Initialization — Deltas

## ADDED Requirements

### Requirement: Init files carry no inline resource sections

An init file SHALL NOT declare inline resource sections (`repositories`,
`packages`, `services`, `files`, `templates`, `directories`, `configuration`).
Under strict decode, an init file carrying any of these keys SHALL fail with
an error naming the key and pointing at blueprint files.

Why: these sections were decoded, validated, and profile-counted but never
applied at runtime — a declaration that silently did nothing. Blueprints are
the single declaration path.

Migration: move the entries into blueprint files under the tree; `rwr convert
--migrate` automates this.

#### Scenario: Init file with an inline packages section

- **WHEN** an init file declares a top-level `packages:` list
- **THEN** initialization fails with an error naming `packages` as an
  unsupported init-file key and pointing at blueprint files
