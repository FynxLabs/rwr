# Omarchy desktop setup

The `omarchy` processor personalizes an installed Omarchy desktop after login.
Run `rwr run omarchy` (or `rwr omarchy`); `rwr all` runs it after ordinary
packages, files, services, Git, scripts and configuration. Explicit init orders
must include `omarchy` after its prerequisites. It never installs or updates
Omarchy itself, runs as the desktop user without sudo, and requires working
Hyprland and Omarchy shell IPC for apply.

```yaml
omarchy:
  - name: desktop
    plugins:
      - id: expose.window-overview
        source:
          git: https://github.com/kristofferR/omarchy-expose.git
        enabled: true
        settings:
          hotCornerEnabled: false
      - id: io.github.woogy7.workspaces
        source:
          git: https://github.com/Woogy7/omarchy-workspace-switcher.git
        enabled: true
        settings:
          minWorkspaces: 5
          maxWorkspaces: 5
        widget:
          visible: false
      - id: omarchy.workspaces
        enabled: true
        widget:
          visible: true
          section: left
          after: omarchy.menu
    shell:
      idle: {screensaver: 300, lock: 360}
      bar: {position: top, transparent: false}
```

Entries support `profiles` and `import`. Import paths and asset sources resolve
relative to the file declaring them, including nested cross-format imports.
Selected entries are combined before applying; conflicting values name both
source files. CUE, YAML, TOML and JSON have the same schema. Static validation
and dry-run work without a running desktop and perform no installation,
network mutation, readiness check, reload, backup or journal write.

## Plugins and configuration

`source` accepts exactly one of `git`, `path`, or `clone`. Git sources use HTTPS
or SSH URLs. An optional `ref` pins a revision. Already-satisfied floating
checkouts are not fetched; `update: true` explicitly refreshes an unpinned Git
source. A changed pin is a desired-state change. All changed sources are staged
and manifests validated before installed resources change.

Plugin IDs come from manifests. A repository name is not an ID. Sources with a
different manifest identity are rejected. Local source directories must contain
regular files rather than symlinks. Omarchy discovers installed plugins under
`~/.config/omarchy/plugins/<id>`; an arbitrary installation directory is not a
supported option.

`source: {clone: omarchy.clock}` explicitly invokes Omarchy's native clone
operation. The declared ID must match its user-derived `<username>.clock` ID.
The native clone initially replaces its source; declare `enabled: false` to
turn it off. No settings or integration operation implicitly clones a built-in.
Packaged plugins have no source and cannot use `state: absent`; disable them
with `enabled: false` instead.

Omitting `enabled` preserves existing activation. `widget.visible` independently
controls bar presence, so an overlay can stay enabled without its optional
widget. Placement accepts `section: left|center|right` and at most one of
`before`, `after`, or zero-based `index`. Missing anchors and conflicting layouts
fail preflight. Built-in replacement operations use the native registry APIs.

`settings` merges declared keys and nested maps into the plugin's existing
entry. Unknown options survive. Empty arrays/maps, false, zero and null are
values. `unset: [nested.option]` explicitly removes a setting. Use nested maps,
not literal dots in option keys; structural identity fields are reserved.

Shell options initially cover idle `screensaver`/`lock` durations, bar
`position`/`transparent`/`centerAnchor`. They are targeted updates, not a complete
replacement of `shell.json`. Existing invalid JSON is an error. Missing user
configuration is seeded from the installed defaults. Writes are atomic, retain
protected prior versions, and reject detected concurrent changes. An external
writer racing the final filesystem rename cannot be locked by RWR; readback and
subsequent status checks expose changes that persist after application.

## Themes, defaults and hooks

```yaml
omarchy:
  - name: appearance
    theme:
      name: my-theme
      source: {path: ../assets/my-theme}
      active: true
      background: ../assets/background.png
    defaults: {browser: firefox, terminal: ghostty, editor: nvim}
    hooks:
      - event: theme-set
        name: 20-personal
        source: ../assets/theme-hook.sh
```

A theme can select stock assets, install from Git, or copy a local directory.
An overlay with a stock theme's slug inherits its packaged files. Git themes
retain their `.git` metadata so native Omarchy theme activation applies its
executable-content restrictions. Theme assets and hooks precede activation;
active name/background are read back afterward.

Defaults require installed applications. Supported names follow the inspected
Omarchy setters: browser `chromium`, `chrome`, `brave`, `brave-origin`, `edge`,
`firefox`, `zen`; terminal `alacritty`, `foot`, `ghostty`, `kitty`; editor `code`,
`cursor`, `zed`, `sublime_text`, `helix`, `vim`, `emacs`, `nvim`. Application
installation belongs in package declarations. The native terminal setter
replaces `~/.config/xdg-terminals.list`; the browser setter updates XDG web
handlers; editor selection updates Omarchy's default editor. Other defaults,
including the agent installer, are not invoked. A notification failure does not
undo a verified persisted default.

Hooks support `theme-set`, `font-set`, `post-boot`, `post-update`, `battery-low`,
and `pre-refresh-pacman`. Each named hook gets an owned wrapper and a protected
copy of its source; arguments are preserved. Unrelated hooks are untouched.
Declared theme-hook execution outcomes are checked because the native runner
can hide failures. Undeclared hooks retain Omarchy's native reporting behavior.

## External screensavers

```yaml
omarchy:
  - name: external-screensaver
    integrations:
      screensaver:
        mode: external
        windowClass: org.omarchy.screensaver
        launch: {exec: /absolute/path/to/engine, args: [start]}
        check: {exec: /absolute/path/to/engine, args: [check]}
```

The stock `omarchy.idle` plugin must be enabled. If a custom idle clone is active,
explicitly restore stock idle in the same blueprint before selecting this route.
The engine is supplied separately. Its check must be nonvisual and return zero
only when ready. Launch/check executables are absolute paths and arguments stay
separate, including quotes and whitespace. Apply never starts a visible
screensaver or locks the desktop.

The current stock idle service invokes `omarchy-launch-screensaver` through
`bash -lc`. RWR installs a narrow dispatcher under
`~/.config/rwr/omarchy/bin` and an identifiable PATH block at the end of the
active Bash login profile (`.bash_profile`, `.bash_login`, or `.profile`). It
checks launcher resolution in a fresh login shell before reporting activation.
This also makes the route available to descendants of those login shells;
existing processes retain their environment. A failed route check restores the
prior profile. No packaged launcher or idle service is edited or cloned.
`RWR_OMARCHY_STOCK_SCREENSAVER=1` selects the absolute packaged fallback.

The engine must use Hyprland window class `org.omarchy.screensaver`, create
windows on the intended monitors, dismiss them on input, and stop when Omarchy
locks. The stock idle service uses those window events to maintain/cancel its
lock timer. RWR checks the installed idle adapter's required source contract
and the engine's readiness command; it cannot prove an arbitrary engine's
visual or locking behavior from an exit code. Test idle trigger, dismissal,
multi-monitor behavior, subsequent locking and stock restoration in a disposable
logged-in session before relying on a new engine.

`mode: stock` explicitly removes RWR's login routing block and verifies that a
fresh login shell resolves the packaged launcher (including an equivalent symlink). Omission leaves routing alone.
Modified or unowned dispatchers/profile blocks are not overwritten. The owned,
dormant dispatcher is retained for later reactivation.

## Keybindings

Keep Lua and helper scripts in ordinary file blueprints. Example assets are in
`examples/omarchy/assets/`. Deploy `rwr-bindings.lua` and executable
`workspace-cycle` into the target user's `~/.config/hypr/`, then include this line
in the `bindings.lua` your blueprint already manages:

```lua
dofile(os.getenv("HOME") .. "/.config/hypr/rwr-bindings.lua")
```

This changes only Ctrl+Alt+Arrow bindings and leaves window-moving shortcuts
alone. The helper reads Workspace Switcher's `maxWorkspaces`, including when
its optional widget is in the bar. Missing settings default to 10; invalid
limits fail without dispatching. No second workspace-count setting exists.
Respect OmaSettings' later override load order. Validate Lua with `luac -p`,
then run `hyprctl reload` and inspect `hyprctl configerrors` after deployment.
These are ordinary file/script declarations, not a second keybinding language.

## Ownership, status and recovery

Each resource has a stable identity such as `plugin/<id>/enabled`,
`plugin/<id>/settings/<path>`, `theme/active`, or `default/browser`. Status probes
actual state even without a prior journal record; unavailable IPC is unknown.
Dry-run lists desired resources without pretending live state was observed.

Created plugin/theme assets and hooks have private ownership/content records
under `~/.local/state/rwr/omarchy`. `state: absent` removes only owned, unchanged
plugins/hooks. Adopted resources and dirty or modified checkouts are retained
with an error. Omitting a declaration never removes it. Ordinary journal entries
contain identities and outcomes, not plugin option maps or whole configuration
files. Previous configuration contents are stored separately with mode 0600.

Generic `rwr uninstall` lists Omarchy as not automatically reversible. Use
explicit desired-state changes to restore settings, stock activation or routing;
RWR does not guess old values or delete shared configuration. Semantic capture
and diff authoring remain follow-up work.

The adapter targets the installed Omarchy plugin catalog/list/validate,
enable/disable/clone/remove, theme and default command contracts, structured
`shell.json`, and Bash-launched stock idle service. Capability failures are
reported before dependent work; unsupported environments get no deferred login
job or second shell process. Automated validation uses disposable files,
command fixtures and the launcher boundary. Live visual desktop acceptance is
separate from those tests.
