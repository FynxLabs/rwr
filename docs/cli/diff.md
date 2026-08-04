# rwr diff

Compare what is actually on the machine against the blueprint tree, and get
the delta as blueprint material - never an apply.

The machine side comes from read-only scans: explicitly-installed packages
per provider (`pacman -Qe`, `apt-mark showmanual`, …), operator-enabled
services, git checkouts, and configs. The tree side is the resolved plan.
Packages the [run journal](../state.md) shows a run applied never report as
hand-added - the tree may have changed since, but you didn't install them.

```bash
# The readable list: + additions (hand-done), - removals (declared, gone)
rwr diff

# One category
rwr diff --packages

# Paste-ready blueprint blocks
rwr diff --packages --format cue

# Route additions into a tree interactively: per group, pick the machine
# file, a Common file the tree imports, or skip
rwr diff --into ~/git/thefynx/rwr-blueprints/Archcraft
```

`--into` needs a terminal; `--format` is the non-interactive path. Routed
edits are written in the destination file's own format - run
`rwr validate`, review, commit. Removals are reported but never
auto-deleted from blueprints.
