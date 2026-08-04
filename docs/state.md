# Run records

Every real run (`rwr all`, `rwr run <processor>`) leaves a record of what it
applied: one JSON file per run under `<configdir>/state/runs/`, plus a
`latest` file naming the newest one. Dry runs write nothing.

The record is evidence, not desired state. Blueprints stay the single source
of what the machine should look like; the record is what `rwr status` uses to
recognize files it wrote and what `rwr uninstall` uses to reverse them.

## Format

One append-only event log: `<configdir>/state/journal.jsonl`. Every line is
a self-contained JSON event; nothing is ever rewritten or mutated.

```json
{"v":2,"kind":"run","id":"20260803-204243-7b0f","started":"...","location":"/path/to/blueprints"}
{"v":2,"kind":"apply","run":"20260803-204243-7b0f","processor":"files","action":"create","identity":{"name":"h.txt","dest":"/home/user/h.txt","sha256":"2d71..."},"outcome":"ok","ok":true}
{"v":2,"kind":"finish","run":"20260803-204243-7b0f","finished":"..."}
{"v":2,"kind":"reverse","run":"20260803-2050-1a2b","processor":"files","identity":{"name":"h.txt","dest":"/home/user/h.txt","sha256":"2d71..."}}
```

- Appends are O(1); a crash loses at most a partial last line, and every
  complete line before it still counts. A run without a `finish` event is
  unfinalized.
- `identity` carries what a later run needs to find the thing again:
  `provider` + `name` for packages, `dest` + `sha256` for files and
  templates, `dest` for directories, `target` for git checkouts, `dir` for
  fonts.
- `rwr uninstall` appends `reverse` events; readers fold them over the
  applies. History is never edited.
- The journal is user-only (`0600`, directory `0700`). Legacy v1 per-run
  record files under `state/runs/` are still read and folded in.

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

Input is the record - never the blueprint tree. With no record it refuses.
The not-reversible list (scripts, configuration writes, users, uploaded SSH
keys, repositories) prints before the confirmation. Removal runs in reverse
apply order; files and git checkouts are hash-/cleanliness-guarded, modified
content is skipped and listed; failures keep going, exit non-zero, and stay
unreversed so a re-run retries them. `--yes` skips the prompt, `--dry-run`
prints the plan and touches nothing.
