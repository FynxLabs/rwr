# Change: Run records, `rwr status`, and `rwr uninstall`

## Why

`rwr all` is a one-way door. There is no inverse operation, no record of what
a run applied, and no desired-vs-actual view short of reading `--dry-run`
logs. Concretely, today:

- The only persisted trace of a run is the empty bootstrap marker file
  (`internal/helpers/bootstrap.go:36-49` creates `<configdir>/bootstrap` with
  no content). Nothing records which packages were installed, which files were
  written, which services were enabled, when, or from which blueprint tree.
- The removal machinery exists but is blueprint-driven, not inverse-driven:
  packages honor `action: remove` (`internal/processors/packages.go:118`) and
  every provider defines a `remove` command (e.g.
  `internal/system/definitions/providers/apt.json` `"remove": "remove -y"`)
  plus `repository.remove` step lists - but the operator must hand-edit
  blueprints to `remove` and re-run, which also loses the declaration.
- Desired state is already resolvable without applying (the TUI's
  `ResolveStage1`, `internal/processors/plan.go:17`, and dry-run centralized
  in `system.RunCommand`), but nothing compares it to the actual machine.

Without a record, "what did rwr do to this machine" and "take it back off"
are both unanswerable - a real cost for the trying-it-out user and for anyone
provisioning short-lived machines.

## What Changes

- **Run record**: every non-dry-run applies get journaled to a state file
  (`<configdir>/state/`, one record per run): timestamp, blueprint tree
  identity (URL/path + commit when git), and per-resource entries - what was
  applied, by which processor, with enough identity to check or reverse it
  (package name + provider, file dest + content hash, service name + action,
  repo name, font name, git target). Recording is best-effort observation:
  a failed run still records what it did apply.
- **`rwr status`**: desired-vs-actual. Resolves the tree (reusing the plan
  machinery), queries actuals per processor - package present via the
  provider's `list`/`search`, file dest exists and hash matches, service
  enabled/running, repo file present - and prints a drift table: in-sync,
  missing, modified, unmanaged-but-recorded. Requires new read-only query
  support per processor; processors that cannot be queried (scripts,
  configuration) report `unknown` honestly rather than guessing.
- **`rwr uninstall`**: reverse of `all`, driven by the run record (not by
  re-reading blueprints, so it works after the tree changed). Reverses in
  reverse application order, per-processor where a true inverse exists:
  - packages → provider `remove`
  - repositories → provider `remove` steps
  - fonts → delete recorded files
  - services → disable/stop units the record created
  - files/directories/git → delete recorded dests, only when the recorded
    content hash still matches (modified files are skipped and listed)
  - **not reversible, and said so**: scripts (arbitrary side effects),
    configuration writes (dconf/plist/registry have no recorded prior value),
    users/groups (destructive; out of scope), ssh keys uploaded to GitHub.
    `rwr uninstall` prints exactly what it will not undo, before doing
    anything.
  - honors `--dry-run`; interactive confirmation by default, `--yes` for
    scripts.

## Breakage

Nothing breaks for existing blueprints. The state file is new and additive;
`rwr all` behavior is unchanged apart from writing the record. Machines
provisioned by older rwr have no record: `rwr status` falls back to
desired-vs-actual only (no "recorded" column) and `rwr uninstall` refuses
with an explanation rather than guessing.

## Impact

- Affected specs: `cli` (two new commands), new capability `state-tracking`
  (run record format and guarantees), touches per-processor specs for the
  query/remove surfaces as they gain them.
- Affected code: new `internal/state` package; `internal/processors/*` gain
  record emission and query hooks; provider definitions already carry the
  remove verbs.
- The record contains resource identities and hashes, no secrets; it lives
  under the config dir with user-only permissions.
