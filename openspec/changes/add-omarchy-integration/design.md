## Context and scope

Omarchy personalization was represented as shell.json snapshots and scripts.
The new processor manages post-login resources while retaining ordinary files,
packages and services for generic provisioning. It does not install Omarchy,
run first-boot setup, replace packaged files, or build a screensaver engine.

## Resource contract

The v1 `omarchy` list contains named entries with profiles and imports. Typed
resources cover plugin sources, activation, settings, widgets, shell options,
themes, application defaults, hooks and screensaver routing. Optional booleans
retain omission. Plugin setting maps remain extensible; false, zero, empty and
null values differ from explicit unset paths.

Selected files and imports resolve to operations with stable identities and
origin paths. Identical declarations deduplicate; nonconflicting plugin presence
and nested settings combine. Conflicts and overlapping setting paths fail
preflight. Paths resolve relative to the declaring file. Planning and execution
share the same operation builder, including profile filtering and counts.

## Discovery and lifecycle

A per-run client injects read probes, argv-based mutations and filesystem roots.
Apply requires the logged-in Linux desktop user, Hyprland IPC, the native plugin
catalog and a valid runtime plugin list. A failed query is unknown state.
Catalog and runtime identities must agree. Filesystem mutation is confined to
the target user's home; external source files are read through bounded roots.

Changed Git/local sources are staged and plugin manifests validated before
installed resources change. Floating sources are not fetched by default. A
changed pin or explicit unpinned update uses content/source ownership guards.
Local assets do not follow symlinks. Native clone operations use the actual
user-derived ID; activation remains a separate explicit declaration. Built-in
files cannot be removed.

Native registry commands handle replacement/clone relationships. Disabling a
built-in widget also records its disabled state, because native widget removal
alone can leave its component loadable. Optional widget visibility remains
independent from overlay/service activation. Widget layouts are reconciled as a
group against all selected anchors; contradictory layouts fail preflight.

## Configuration and ownership

Settings are targeted structured edits with unknown fields preserved. Missing
user config is seeded from installed defaults. Invalid JSON fails. Atomic
writes compare against the observed content, preserve modes, keep protected
backups and verify readback. Detected concurrent writes fail cleanly; no claim is
made that an RWR lock can exclude a GUI writer racing the final rename.

Private ownership records distinguish created resources from adopted assets.
Explicit removal rejects adopted or modified content and changed Git origins.
Runtime convergence waits briefly for shell hot-reload. Partial failures block
dependent actions and produce a terminal result for each planned resource.
Generic uninstall explicitly reports Omarchy operations as not automatically
reversible; restoration uses declared state rather than guessing old values.

## Themes, defaults and hooks

Local overlays retain stock inheritance. Git themes retain `.git` metadata so
native activation enforces its executable-content restrictions. Assets and
hooks precede activation. Defaults use the native browser, terminal and editor
setters after checking applications; persistence is verified independently of
notification delivery. Native setter breadth is documented.

Managed hooks have protected payloads and small argument-preserving wrappers.
Wrappers record a run nonce and exit status. Theme activation verifies outcomes
for declared hooks because the native runner otherwise hides their failures.
Unrelated hooks retain native reporting behavior.

## Screensaver adapter

The inspected idle service calls a fixed launcher through `bash -lc`; placing a
file in ~/.local/bin alone does not establish precedence. RWR installs a narrow
user-owned dispatcher and a marked PATH block at the end of the effective Bash
login profile, then probes launcher resolution in a fresh login shell. Failed
resolution restores the prior profile and dispatcher. Stock mode removes the
block and verifies the packaged launcher, allowing equivalent symlinks.

Launch/check declarations use absolute executables and separate arguments. The
nonvisual check runs before activation. The dispatcher only routes execution;
it does not copy or clone stock idle logic. The external engine must use
`org.omarchy.screensaver`, implement dismissal and stop when locked. Static
capability checks verify the inspected stock lifecycle boundary; they do not
prove an arbitrary engine's visual or locking behavior. Controlled live idle,
dismissal, multi-monitor and lock acceptance remains necessary for a supplied
engine. Development/tests do not mutate the operator's desktop.

## Integration and validation

The processor is registered in CLI, directory/content routing, strict schemas,
profiles, imports, stage-one diagnostics, stage-two resources, execution,
reporting, status and explicit uninstall limitations. Status carries private
desired-state data without putting option maps into ordinary journal entries.
Dry-run performs no runtime probes or mutations and labels live state unknown.

Tests use disposable files and command fixtures for repeat application, source
identity checks, ownership, settings, placement, cancellation/discovery failure,
hook outcome verification, shell argument fidelity and routing recovery.
Examples cover JSON, YAML, TOML and CUE. The optional binding module and cycling
helper remain ordinary file assets; tests cover the single maxWorkspaces source
of truth and missing-setting fallback. Semantic capture/diff authoring is a
separate follow-up.
