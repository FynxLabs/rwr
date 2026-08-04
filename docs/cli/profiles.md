# Profile CLI Commands

This page is the command-line reference for profiles in RWR. Profiles let you
select which items of a blueprint tree apply to the machine you are on. For the
model itself, read the [Profile System Overview](../profiles.md).

## Profile CLI Overview

There are two pieces:

1. **The `--profile` / `-p` flag**, which chooses the profiles that are active
   for a run.
2. **The `rwr profiles` command**, which reads your blueprints and tells you
   which profile names they declare.

## The `--profile` / `-p` flag

`--profile` is a global flag, so it is accepted by every command. It only
changes what is applied when a command applies something - that is `rwr all` and
`rwr run <processor>`.

> [!IMPORTANT]
> `rwr --profile work` applies nothing. Bare `rwr` prints the help text. The
> command that applies a profile is `rwr all --profile work`, or
> `rwr run <processor> --profile work`.

### Syntax

```bash
rwr all --profile PROFILE1,PROFILE2
rwr all -p PROFILE1 -p PROFILE2
```

Both forms may be mixed; the values accumulate.

### Single profile

```bash
rwr all --profile work
rwr all -p work

rwr run packages --profile work
rwr run services -p gaming
```

### Multiple profiles

```bash
# Comma-separated
rwr all --profile work,gaming,dev

# Repeated flag
rwr all -p work -p gaming -p dev

# Mixed
rwr all --profile work,gaming -p dev
```

### The `all` keyword

`all` is the one reserved profile name. When it is among the active profiles,
every item applies whatever its `profiles` field says.

```bash
rwr all --profile all

# "all" wins over the others: this is the same as --profile all
rwr all --profile work,gaming,all
```

### No `--profile` at all

```bash
rwr all
```

> [!IMPORTANT]
> This applies **everything**, not only the items without a `profiles` field.
> With no active profiles the filter is skipped entirely, so profile items apply
> too. Filtering starts as soon as you name at least one profile. If you want
> only the base items, there is no flag for that today.

### Commands that ignore the flag

`--profile` is accepted by `rwr validate`, `rwr profiles`, `rwr config` and
`rwr version` because it is a global flag, but none of them read it.
`rwr validate` checks every blueprint file in the tree regardless of profiles.

## `rwr profiles`

`rwr profiles` walks the blueprint tree named by your init file and reports every
profile its entries declare, with a count of the entries carrying each one.

```bash
rwr profiles
rwr profiles --init-file path/to/init.yaml
```

### Example output

```text
Available profiles (2 found):

  • dev (1 items)
  • work (2 items)

  base items (always applied): 2

Usage examples:
  rwr all --profile dev
  rwr all --profile dev --profile work
  rwr run packages --profile dev,work
  rwr all --profile all
```

When the tree declares no profiles at all:

```text
No profiles found in your blueprints.
All 12 item(s) are base items and always apply.
```

### What it counts

* One count per entry carrying a profile, per profile. An entry listing two
  profiles is counted once under each.
* `base items` is the number of entries with no `profiles` field.
* Entries written inline in the init file are counted as well as the ones in the
  blueprint tree.

The command finds a blueprint's type from the directory it sits in, the same way
a run does, so a file has to be under `packages/`, `services/`, `files/` and so
on to be read. It only reads files whose extension matches the `format` declared
in the init file.

## Debugging profile selection

`--debug` reports, per blueprint file, how many entries survived the filter:

```bash
rwr all --debug --profile work
```

```text
DEBU Filtering packages: 3 total, 3 matching active profiles [work]
DEBU Filtering services: 1 total, 1 matching active profiles [work]
```

There is no per-item log line saying why one entry was kept and another dropped.

Combine with `--dry-run` to see exactly which commands a profile combination
would run, without running them:

```bash
rwr all --dry-run --profile work,gaming
```

## Error handling

RWR does **not** check the names you pass against the ones your blueprints
declare. A misspelled profile is not an error and produces no warning - it simply
matches nothing, and the run applies the base items only. If a run installs less
than you expected, check the spelling against `rwr profiles`; the names are
case-sensitive.

## Command reference

| Flag | Description | Example |
|------|-------------|---------|
| `--profile PROFILES` | Comma-separated list of profiles to activate | `--profile work,gaming` |
| `-p PROFILE` | Short form; repeat it for more than one | `-p work -p gaming` |

| Command | Description |
|---------|-------------|
| `rwr profiles` | List the profiles the blueprint tree declares |
| `rwr all --profile ...` | Apply every blueprint with those profiles active |
| `rwr run <processor> --profile ...` | Apply one processor with those profiles active |

## Examples by use case

### Developer workflow

```bash
# See what the tree offers
rwr profiles

# Frontend environment
rwr all --profile frontend

# Full stack
rwr all --profile frontend,backend,database

# Check first
rwr all --dry-run --debug --profile dev
```

### System administration

```bash
rwr all --profile server
rwr all --profile desktop,productivity
rwr all --profile all
```

### One processor at a time

```bash
rwr run packages --profile work
rwr run services --profile gaming
rwr run files --profile work,dev
```

`rwr run` takes exactly one processor name. There is no `rwr run all` and no
comma-separated list of processors; use `rwr all` to run everything.

## Best practices

1. **Use `-p` interactively, `--profile` in scripts.** The long form reads
   better in automation.
2. **Group related profiles in one run**: `rwr all --profile work,dev` rather
   than two commands.
3. **Check with `--dry-run` first.** Nothing validates a profile name for you,
   so a dry run is how a typo gets caught.
4. **Run `rwr profiles` in a tree you did not write** before you run anything
   else.

## Related Documentation

* [Profile System Overview](../profiles.md) - Complete profile system documentation
* [CLI Commands & Flags](command-and-flags.md) - General CLI documentation
* [Profile Best Practices](../profile-best-practices.md) - Organizational guidelines
* [General Best Practices](../best-practices.md) - Overall RWR best practices
