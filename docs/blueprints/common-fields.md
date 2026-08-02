# Fields Common to Every Blueprint

This page describes the settings that behave the same way in every blueprint
type, and the one decoding rule that applies to all of them.

## Unknown keys are an error

> [!IMPORTANT]
> Blueprints decode strictly, in YAML, JSON and TOML alike. A key that the
> blueprint type does not define stops the run. Examples: a misspelled
> `pacakges:`, or `profile:` written for `profiles:`. The error names the file
> and, for YAML, the line.

Earlier versions dropped an unknown key in silence. A misspelled section then
produced an empty list, every processor found nothing to do, and the run
reported success although it changed nothing. A misspelled `profiles` was worse: the entry lost its
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

`import` names another blueprint file to pull entries from. RWR resolves the
path relative to your blueprint directory. You can nest imports, and RWR detects
and refuses circular imports. An entry that carries an `import` carries
nothing else.

Supported by: `packages`, `repositories`, `files`, `templates`, `directories`,
`git`, `scripts`, `services`, `ssh_keys`, `users` and `groups`.

**Not** supported by `fonts` or `configurations` — those two types have no
`import` field, so an `import` key in them is now a decode error rather than a
silently ignored one.

## `interactive`

`interactive: true` or `interactive: false` overrides the global `--interactive`
flag for a single entry. Omit it to follow the flag.

Supported by `files`, `templates`, `directories`, `packages`, `repositories`,
`git`, `scripts`, `services`, `ssh_keys` and `users`. Not supported by `fonts`,
`groups` or `configurations`.

## `name` and `names`

Every type has `name`. A `names` list, which repeats the rest of the entry once
per name, is read by `files`, `templates`, `packages` and `fonts`.

`directories` and `configurations` accept a `names` key and do **not** read it —
write one entry per item there. Every other type rejects `names` outright.
