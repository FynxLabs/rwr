# CLI — Deltas

## ADDED Requirements

### Requirement: rwr capture generates a blueprint tree from the machine

`rwr capture` SHALL scan the machine, present the findings as a per-category
selection form (packages per provider explicit-only and pre-selected;
configs pre-selected only for known dotfiles; services unselected by
default; git checkouts pre-selected), and generate a blueprint tree in the
chosen registry format from the selection — init file, per-category
blueprints, and selected config files copied under `files/src/` with
matching entries. `--all` SHALL take the defaults without the form;
`--manifest` SHALL add a root manifest carrying the machine's detected
matchers.

#### Scenario: Handcrafted machine to tree

- **GIVEN** a machine with explicit pacman and cargo packages and known
  dotfiles
- **WHEN** `rwr capture --all --format cue out/` runs
- **THEN** out/ contains an init and packages blueprints per provider and
  the dotfiles under `files/src/` with file entries

#### Scenario: Dependency noise never captured

- **WHEN** the packages page is built for a provider with `list_explicit`
- **THEN** only explicitly-installed packages appear on it

#### Scenario: Secret-shaped paths need explicit intent

- **GIVEN** the config scan finding `~/.ssh/config` and key material
- **WHEN** the selection defaults are computed
- **THEN** key material is never pre-selected, and selecting it warns

### Requirement: A capture that emits an invalid tree fails

`rwr capture` SHALL run validation over the generated tree and SHALL exit
non-zero when it does not pass, naming the failures. A generator whose
output needs hand-repair before first use has not captured anything.

#### Scenario: Broken emission is loud

- **WHEN** the generated tree fails strict decode
- **THEN** capture exits non-zero naming the failing file
