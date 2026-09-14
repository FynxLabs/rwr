# Omarchy configuration tool

Choose one format directory and use its init file. Each example uses a
`configurations` entry with `tool: omarchy` to install Exposé and Workspace
Switcher, disable Exposé's hot corner, configure a 1–5 workspace ring, hide
Workspace Switcher's optional bar widget, and keep Omarchy's stock workspace
widget. It does not enable every plugin previously tried on a machine.

The sibling `assets` directory contains an optional user-owned binding module
and executable workspace-cycle helper for normal file blueprints. Include the
module from the bindings file your blueprint already manages; the examples do
not overwrite an existing bindings.lua or edit the live desktop during tests.

See [the resource documentation](../../docs/blueprints/omarchy.md) for sources,
clones, themes, default applications, hooks, screensaver replacement, ownership,
status and post-login acceptance checks.
