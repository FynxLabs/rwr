## ADDED Requirements

### Requirement: Reconcile declared desktop resources
RWR SHALL provide an omarchy processor supporting plugin sources and explicit clones, activation, settings, widget placement, themes, application defaults, hooks and external screensaver routing. It SHALL preserve undeclared state and reject conflicting selected declarations before mutation.

#### Scenario: Repeat a setup
- **WHEN** the declared state is already present
- **THEN** RWR reports unchanged resources without reinstalling plugins or rewriting configuration

### Requirement: Respect runtime and ownership boundaries
RWR SHALL require the selected resources' desktop capabilities, avoid privileged execution, preserve packaged files, and prevent removal of adopted or subsequently modified assets. Dry-run SHALL perform no mutation. Status SHALL report unavailable observations as unknown.

#### Scenario: Desktop is unavailable
- **WHEN** shell discovery fails
- **THEN** apply fails before changing the desktop rather than treating the plugin list as empty

### Requirement: Integrate external screensavers without cloning idle
An explicit screensaver integration SHALL use a verified user-owned launch route, retain stock idle and lock behavior, require readiness and window-lifecycle compatibility, and support restoration to stock.

#### Scenario: External engine cannot pass readiness
- **WHEN** its declared check fails
- **THEN** the active launch route remains unchanged
