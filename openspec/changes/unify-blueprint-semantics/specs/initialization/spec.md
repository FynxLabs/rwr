# Initialization — Deltas

## REMOVED Requirements

### Requirement: Init-file inline resource sections

The init file's inline resource sections (`repositories`, `packages`,
`services`, `files`, `templates`, `directories`, `configuration`) are removed
from the schema. They were decoded, validated, and profile-counted but never
applied at runtime — a declaration that silently did nothing.

Blueprints are the single declaration path. Under strict decode, an init file
still carrying these keys is now an error naming them, instead of a silent
no-op.

Migration: move the entries into blueprint files under the tree; a future
`rwr convert` change automates this.

#### Scenario: Init file with an inline packages section

- **WHEN** an init file declares a top-level `packages:` list
- **THEN** initialization fails with an error naming `packages` as an
  unsupported init-file key and pointing at blueprint files
