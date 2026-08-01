# Blueprint schema versioning

Blueprints declare which version of the schema they are written in, so RWR can
change a blueprint format without silently misreading configuration that was
written against the old one.

## Versions are per blueprint type

There is no single schema number covering everything. Each blueprint type —
`packages`, `files`, `services`, and so on — carries its own version.

That is deliberate. If one shared number covered the whole schema, a breaking
change to `packages` would move `files`, `git`, `scripts` and everything else to
v2 as well, even though none of them changed. Do that a few times and you are on
v11 with no way to tell which resource actually changed at each step.

With per-type versions, a breaking change to `packages` produces exactly one
change: `packages` is now v2. Everything else is still v1, and blueprints for
those types keep working untouched.

## Declaring a version

**In the init file**, applying to the whole tree:

```yaml
blueprints:
  format: yaml
  location: "."
  schema_version: 1
```

**In an individual blueprint file**, overriding the tree for that file only:

```yaml
schema_version: 2
packages:
  - name: git
    action: install
```

The most specific declaration wins:

1. the blueprint file's own `schema_version`
2. the init file's `schema_version`
3. v1

## Nothing declared means v1

A blueprint that declares no version is read as v1. Every blueprint written
before versioning existed is v1, so absence has to mean 1 rather than "unset" —
otherwise adding versioning would have broken every existing tree.

You never have to add `schema_version` to keep working. You add it when you want
a version that is not v1.

## Migrating one resource type

When `packages` gains a v2, a tree can move one file at a time:

```yaml
# init.yaml — the tree is still v1
blueprints:
  schema_version: 1
```

```yaml
# packages/dev-tools.yaml — this file is v2
schema_version: 2
packages:
  - name: git
    action: install
```

```yaml
# packages/base.yaml — still v1, still works
packages:
  - name: curl
    action: install
```

Both files are read correctly in the same run. There is no flag day.

## Unsupported versions are an error

A blueprint asking for a version this build does not implement fails, naming the
version it wanted:

```
packages: schema version 2 is not supported by this build (supports 1) —
upgrade rwr, or write this blueprint in a supported version
```

This is intentionally a hard failure. RWR installs packages, writes files and
runs scripts as root; reading a v2 blueprint as though it were v1 would mean
acting on a misunderstanding of what the file asked for. Refusing is the safer
outcome.

A tree-wide version has to be supported by *every* blueprint type, since it
applies to all of them. If only some types have a v2, declare it per file
instead — the error says so.

## Adding a version (for contributors)

Each version of a blueprint type is its own struct describing exactly what that
version accepts on disk. Decoding picks the struct by resolved version and then
converts it to the canonical type the processors consume, so nothing past the
decoder knows more than one version exists.

To add a v2 for a type, in `internal/types/`:

1. Define the struct for the new wire format, e.g. `packagesV2`.
2. Give it a `Canonical()` that maps it onto `PackagesData`.
3. Register it in `schemaRegistry` under `{BlueprintTypePackages: {2: ...}}`.
4. Add `2` to that type's entry in `supportedSchemaVersions`.

Nothing else changes. The processors keep consuming `PackagesData` and no other
blueprint type is affected.
