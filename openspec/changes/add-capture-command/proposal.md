# Change: rwr capture — turn a handcrafted machine into a blueprint tree

Depends on: add-system-scan (the harvest layer).

## Why

The first blueprint tree is the expensive one. An operator with a machine
they have shaped over years — packages across pacman *and* cargo *and* npm,
a decade of dotfiles, hand-enabled services — has no path into rwr except
transcribing it all by hand. That is exactly the work a tool that already
knows every provider's package list, the blueprint schema, and four output
formats should do.

The trap is dumping `pacman -Q`: a capture that includes everything the OS
ever installed is as useless as no capture. The scan layer's explicit-only
filtering handles packages; for configs and services no heuristic beats a
human with a checklist, so the human gets a checklist.

## What Changes

- `rwr capture [dir]`: scans the machine (add-system-scan) and walks the
  operator through a multi-page huh form — one page per category:
  - packages (per provider, explicit-only, pre-selected);
  - configs (dotfiles + `~/.config` survivors of the noise list,
    pre-selected only for the known-dotfile set);
  - services (enabled units, unselected by default — most are distro
    plumbing);
  - git checkouts (discovered clones, pre-selected).
- Generates into `dir` (default `./rwr-blueprints`):
  - a tree in the chosen format (`--format cue|yaml|json|toml`, default
    cue): init file, per-category blueprint files, selected config files
    copied under `files/src/` with matching file entries;
  - `--manifest` also writes a root manifest with the current machine's
    OS/distro matchers, ready to grow more machines.
- The generated tree SHALL pass `rwr validate` before capture reports
  success — a capture that emits an invalid tree has failed, loudly.
- Non-interactive mode (`--all`, CI/scripting) takes every pre-selected
  default without the form.

## Breakage

None. New command; nothing existing changes.

## Impact

- Affected specs: `cli` (new command), reads `system-scan`.
- Affected code: `cmd/capture.go` (wiring), `internal/capture` (form +
  generation), scan emission layer for the file contents.
