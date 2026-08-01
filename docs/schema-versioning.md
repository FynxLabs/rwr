# Blueprint schema versioning

A blueprint can give the version of the schema that it uses. RWR can then change
a blueprint format and still read your older blueprints correctly.

## Each blueprint type has its own version

There is no single version number for the full schema. Each blueprint type has a
version. The types are `packages`, `files`, `services`, and the others.

This is intentional. With one version number for the full schema, a change to
`packages` moves `files`, `git`, `scripts` and all other types to v2. Those types
did not change. After a small number of such changes, the schema is at v11, and
the version number does not tell you which type changed.

With a version for each type, a change to `packages` makes one change:
`packages` is v2. All other types stay at v1. Blueprints for those types continue
to operate.

## How to give a version

In the init file, for the full tree:

```yaml
blueprints:
  format: yaml
  location: "."
  schema_version: 1
```

In one blueprint file, for that file only:

```yaml
schema_version: 2
packages:
  - name: git
    action: install
```

RWR uses the most specific version:

1. the `schema_version` in the blueprint file
2. the `schema_version` in the init file
3. the latest version that RWR supports for that blueprint type

## No version means the latest version

RWR reads a blueprint that gives no version as the latest version for that type.

A new blueprint then uses the current schema, and you do not write a version. Add
a version when you want RWR to hold the blueprint at that version.

There is a result to know. A blueprint with no version moves to a new schema when
you upgrade RWR. Give a version if the tree must stay at one version. Two
examples are a tree that is not ready for a new format, and a tree that must
operate in the same way on each machine.

```yaml
blueprints:
  schema_version: 1
```

That line holds the full tree at v1. A blueprint file can still give a later
version.

The latest version is different for each type. When `packages` moves to v2, RWR
reads a `packages` blueprint with no version as v2. RWR reads a `files`
blueprint as v1, because `files` did not change.

## How to move one type to a new version

When `packages` moves to v2, a tree can move one file at a time:

```yaml
# init.yaml — the tree is at v1
blueprints:
  schema_version: 1
```

```yaml
# packages/dev-tools.yaml — this file is at v2
schema_version: 2
packages:
  - name: git
    action: install
```

```yaml
# packages/base.yaml — still at v1, and it operates
packages:
  - name: curl
    action: install
```

RWR reads both files correctly in the same run. You do not migrate the full tree
on one day.

## An unsupported version stops the run

RWR stops when a blueprint gives a version that this build cannot read. The error
gives the version:

```
packages: schema version 2 is not supported by this build (supports 1) —
upgrade rwr, or write this blueprint in a supported version
```

CAUTION: This error stops the run. RWR installs packages, writes files, and runs
scripts as root. A v2 blueprint that RWR reads as v1 can do damage to the
machine.

A version in the init file applies to each blueprint type. All types must support
that version. Give the version in the blueprint file when only some types have
the new version.

## How to add a version (for contributors)

Each version of a blueprint type is a separate struct. The struct gives the
fields that the version accepts. RWR selects the struct with the version, then
converts the data to the type that the processors use. Code after the decoder
reads one type only.

To add a v2 for a type, in `internal/types/`:

1. Write the struct for the new format. An example is `packagesV2`.
2. Add a `Canonical()` method that converts the struct to `PackagesData`.
3. Add the struct to `schemaRegistry`, at `{BlueprintTypePackages: {2: ...}}`.
4. Add `2` to the entry for that type in `supportedSchemaVersions`.

Make no other changes. The processors continue to read `PackagesData`, and the
other blueprint types do not change.
