# Run records

Every real run (`rwr all`, `rwr run <processor>`) leaves a record of what it
applied: one JSON file per run under `<configdir>/state/runs/`, plus a
`latest` file naming the newest one. Dry runs write nothing.

The record is evidence, not desired state. Blueprints stay the single source
of what the machine should look like; the record is what `rwr status` uses to
recognize files it wrote and what `rwr uninstall` uses to reverse them.

## Format

```json
{
  "recordVersion": 1,
  "id": "20260803-094124-3bc1ebd2",
  "started": "2026-08-03T09:41:24-05:00",
  "finished": "2026-08-03T09:41:28-05:00",
  "finalized": true,
  "location": "/path/to/blueprints",
  "entries": [
    {
      "processor": "files",
      "action": "create",
      "identity": {
        "name": "hello.txt",
        "dest": "/home/user/hello.txt",
        "sha256": "8f4343…"
      },
      "outcome": "ok",
      "ok": true
    }
  ]
}
```

- The file is rewritten after every entry: a crash mid-run leaves a valid,
  readable record with `finalized: false`.
- `identity` carries what a later run needs to find the thing again:
  `provider` + `name` for packages, `dest` + `sha256` for files and
  templates, `dest` for directories, `target` for git checkouts, `dir` for
  fonts.
- Entries are never deleted. `rwr uninstall` marks what it removed with
  `"reversed": true`; the journal is append-only history.
- Records are user-only (`0600`, directory `0700`). A record with a newer
  `recordVersion` than the binary understands is refused, not misread.

## `rwr status`

Read-only, never elevates, exits 1 on drift. Classes per item:

| Class | Meaning |
|-------|---------|
| `in-sync` | present and (where a hash is recorded) unmodified |
| `missing` | desired but not found |
| `modified` | found, but content differs from the recorded apply |
| `unknown` | honestly not queryable (scripts, configuration, users, ssh_keys, repositories; or no usable provider list) |
| `stale` | recorded by a past run, no longer in the tree |

## `rwr uninstall`

Input is the record — never the blueprint tree. With no record it refuses.
The not-reversible list (scripts, configuration writes, users, uploaded SSH
keys, repositories) prints before the confirmation. Removal runs in reverse
apply order; files and git checkouts are hash-/cleanliness-guarded, modified
content is skipped and listed; failures keep going, exit non-zero, and stay
unreversed so a re-run retries them. `--yes` skips the prompt, `--dry-run`
prints the plan and touches nothing.
