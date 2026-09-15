## Why

Omarchy setup currently relies on whole-file shell snapshots and scripts. Add native, repeatable post-login desktop resources that preserve undeclared state.

## What Changes

Add `omarchy` as a tool in the existing `configurations` blueprint for plugin
sources, activation, settings, placement, themes, defaults, hooks, and an
explicit external screensaver route. Include strict validation, planning,
status, reporting, ownership safeguards and examples. Do not add a standalone
blueprint type, processor, run target, or init-order entry.

## Capabilities

### New Capabilities
- `omarchy-desktop`: Declarative post-login desktop reconciliation.

### Modified Capabilities

## Impact

The configuration schema and processor pipeline, status, uninstall, examples,
and documentation. No packaged Omarchy files or live desktop are modified
during implementation.
