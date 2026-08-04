# Design: Run records, status, uninstall

## Context

rwr is convergence-oriented (`openspec/config.yaml`: running `rwr all` twice
is the normal case) but write-only: nothing records what a run did. This
design adds an observation journal and two consumers of it. It deliberately
does NOT turn rwr into a state-enforcing system like Terraform - blueprints
remain the source of desired state; the record is evidence of past applies,
never an input to `all`.

## Goals

- `rwr all` leaves a truthful record of what it changed.
- `rwr status` answers "does this machine match the tree" without applying.
- `rwr uninstall` reverses what is honestly reversible and enumerates what is
  not, before touching anything.

## Non-goals

- Rollback of file *contents* (no prior-value capture; that is a backup tool).
- Reversing scripts, configuration writes, users/groups, or uploaded SSH keys.
- Using the record to skip work during `all` (convergence already handles it).
- Cross-machine or remote state.

## Decisions

### Record format and location

- `<configdir>/state/runs/<timestamp>-<shortid>.json`, plus `latest` pointer.
  JSON, versioned (`recordVersion: 1`), written incrementally during the run
  (crash leaves a truthful partial record, marked unfinalized).
- Per-resource entry: `{processor, action, identity, detail, ok}` where
  `identity` is enough to find the thing again: `{provider, name}` for
  packages, `{dest, sha256}` for files/templates, `{name, action}` for
  services, `{name}` for repositories/fonts, `{target}` for git.
- Only *applies* are recorded (dry-run writes nothing). Removal runs append
  removal entries; `uninstall` finalizes by marking entries reversed rather
  than deleting records - the journal is append-only history.
- Emission point: the processors already funnel failures through the ledger
  (`internal/processors/failures.go`); record emission rides the same
  per-item points, so coverage is per-item, not per-processor-exit.

### `rwr status`

- Desired side: reuse blueprint resolution (the `ResolveStage1` /
  `plan_stage2` machinery, `internal/processors/plan.go`) - no new parser.
- Actual side: a new optional per-processor `Query` capability:
  - packages: provider `list` command output (already defined per provider,
    e.g. apt's `dpkg --get-selections`) parsed for presence
  - files/templates: dest exists + sha256 comparison against rendered content
  - services: platform query (`systemctl is-enabled/is-active`, launchctl,
    sc.exe)
  - repositories: source-file presence at the provider's `paths`
  - fonts/git: recorded-path existence
  - scripts, configuration, users, ssh_keys: report `unknown` - a query that
    cannot be honest is worse than none
- Output rows: `in-sync | missing | modified | unknown`; with a record
  available, an extra class `stale` (recorded but no longer in the tree).
  Exit code 0 when everything in-sync/unknown, 1 on drift - scriptable.
- Never elevates and never mutates: queries run unelevated, read-only.

### `rwr uninstall`

- Input is the record (union of unreversed entries across runs), not the
  blueprint tree - uninstall after editing or deleting the tree still works.
  No record → refuse with an explanation (guessing at removals from a tree
  that may have changed is how you delete the wrong file).
- Reverse application order (the inverse of the `all` processor order,
  `internal/processors/all.go`), per-item within each processor.
- Safety rails:
  - files/dirs/git: delete only when the recorded sha256 still matches the
    disk content; modified files are skipped and listed
  - packages: provider `remove` verb; a package the record shows applied but
    `status` shows absent is skipped silently (already gone)
  - never touch anything without a record entry
  - the not-reversible list (scripts, configuration, users, ssh uploads) is
    printed up front; `--dry-run` shows the full plan via the existing
    central dry-run path in `system.RunCommand`
  - interactive confirm by default; `--yes` bypasses
- Partial failure: keep going per-item, ledger the failures, exit non-zero,
  leave failed entries unreversed so a re-run retries them.

## Risks

- Provider `list` output parsing differs per provider; scope: presence only,
  not versions, and add per-provider parse fixtures.
- Record/actual divergence (operator removed things by hand): `status` is the
  honest surface for that; `uninstall` skips already-gone items.
- Elevation: removal needs the same elevation the apply needed; entries record
  whether the apply was elevated so uninstall can request it up front.
