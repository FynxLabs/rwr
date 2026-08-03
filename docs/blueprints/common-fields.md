# Fields Common to Every Blueprint

This page describes the settings that behave the same way in every blueprint
type, and the one decoding rule that applies to all of them.

## Unknown keys are an error

> [!IMPORTANT]
> Blueprints decode strictly, in YAML, JSON and TOML alike. A key the blueprint
> type does not define — a misspelled `pacakges:`, a `profile:` that should have
> been `profiles:` — stops the run with an error naming the file and, for YAML,
> the line.

An unknown key used to be dropped in silence. A misspelled section then produced
an empty list, every processor found nothing to do, and the run reported success
having changed nothing. A misspelled `profiles` was worse: the entry lost its
scoping and ran on every machine. Both are now reported.

`rwr validate` decodes the same way, so it reports the same error before
anything is applied.

## `profiles`

Every blueprint type supports `profiles`: a list of profile names the entry
belongs to.

An entry with no `profiles` is a base item and is always processed. An entry
with `profiles` is processed only when one of those profiles is active
(`rwr all --profile dev`). See [Profiles](../profiles.md).

```yaml
packages:
  - name: git         # base item, always installed
    action: install

  - name: docker
    action: install
    profiles:
      - dev
      - work
```

## `import`

`import` names another blueprint file to pull entries from. The path is resolved
relative to your blueprint directory, imports may be nested, and circular
imports are detected and refused. An entry that carries an `import` carries
nothing else.

Supported by: `packages`, `repositories`, `files`, `templates`, `directories`,
`git`, `scripts`, `services`, `ssh_keys`, `users` and `groups`.

**Not** supported by `fonts` or `configurations` — those two types have no
`import` field, so an `import` key in them is now a decode error rather than a
silently ignored one.

## `interactive`

`interactive: true` or `interactive: false` overrides the global `--interactive`
flag for a single entry. Omit it to follow the flag.

Read per entry by `directories`, `packages`, `repositories`, `scripts`,
`services`, `ssh_keys` and `users`. `files`, `templates` and `git` accept the
key but do **not** read it — those processors follow only the global flag. Not
supported at all by `fonts`, `groups` or `configurations`.

## `schema_version`

Every blueprint file may declare a `schema_version` at the top level, next to
its entry list:

```yaml
schema_version: 1
packages:
  - name: git
    action: install
```

A file's declaration overrides the tree-wide version from the init file, which
is how a single blueprint can move to a newer schema while the rest of the tree
stays put. Today every blueprint type supports only version `1`; declaring a
version a type does not support is an error.

## `name` and `names`

Every type has `name`. A `names` list, which repeats the rest of the entry once
per name, is read by `files`, `templates`, `packages` and `fonts`.

`directories` and `configurations` accept a `names` key and do **not** read it —
write one entry per item there. Every other type rejects `names` outright.
