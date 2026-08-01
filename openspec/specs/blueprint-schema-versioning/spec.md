# Blueprint Schema Versioning Specification

## Purpose

RWR reads blueprints written by operators who upgrade the binary on their own
schedule. This capability defines how a blueprint declares the schema it is written
in, so a format change can land without invalidating trees that already work.

## Requirements

### Requirement: Each blueprint type carries its own version

RWR SHALL track a schema version per blueprint type, not one version for the whole
schema.

A breaking change to `packages` SHALL move `packages` to v2 and leave `files`, `git`,
`scripts` and every other type at v1.

A single version number for the whole schema drags unchanged types forward on every
change, arrives at v11 over a handful of edits, and stops telling anyone which type
actually changed.

#### Scenario: One type changes

- **WHEN** `packages` gains a v2 wire format
- **THEN** `packages` supports v1 and v2
- **AND** every other blueprint type still supports v1 only
- **AND** existing `files` blueprints continue to read unchanged

### Requirement: The most specific declaration wins

RWR SHALL resolve a blueprint's schema version in this order:

1. the `schema_version` declared in the blueprint file
2. the `schema_version` declared for the tree in the init file
3. the latest version this build supports for that blueprint type

A file-level declaration SHALL override the tree-wide one. That is what makes a
single-resource migration possible: move one `packages` file to v2 and leave the
rest of the tree alone.

#### Scenario: A file overriding the tree

- **WHEN** the init file declares `schema_version: 1` for the tree
- **AND** one packages blueprint declares `schema_version: 2`
- **THEN** that file is read as v2 and every other file as v1, in the same run

#### Scenario: A tree-wide declaration

- **WHEN** the init file declares `schema_version: 1` and no file declares a version
- **THEN** every blueprint is read as v1

### Requirement: No declaration means the latest version

RWR SHALL read a blueprint that declares no version as the latest version it
supports for that type.

A blueprint written today then gets today's schema with no boilerplate, and the
version field becomes what an operator adds to pin rather than what they must add to
start.

The consequence is that an undeclared blueprint follows the schema forward across
upgrades. A tree that must stay on one version SHALL say so, which is what the
declaration is for.

#### Scenario: A new blueprint with no version field

- **WHEN** a packages blueprint declares no `schema_version`
- **AND** the latest supported version for `packages` is 2
- **THEN** it is read as v2

#### Scenario: Latest differs per type

- **WHEN** `packages` is at v2 and `files` is at v1
- **AND** neither blueprint declares a version
- **THEN** the packages blueprint is read as v2 and the files blueprint as v1

### Requirement: An unsupported version stops the run

RWR SHALL refuse a blueprint that declares a version this build cannot read, and
SHALL report the requested version, the type, and the versions this build supports.

RWR installs packages, writes files, and runs scripts with elevation. A v2 blueprint
misread as v1 does the wrong thing to somebody's machine, which is worse than
refusing.

A tree-wide declaration applies to every blueprint type, so RWR SHALL reject a
tree-wide version that any type cannot read, and SHALL say which types cannot.

#### Scenario: A blueprint from a newer RWR

- **WHEN** a packages blueprint declares `schema_version: 2` and this build supports
  only v1
- **THEN** the run stops with an error naming the type, the requested version, and
  the supported versions

#### Scenario: A tree-wide version no type supports

- **WHEN** an init file declares `schema_version: 2` and no type supports v2
- **THEN** the run stops and the error names the unsupported types and suggests
  declaring the version per file instead

### Requirement: The wire format and the processor type are separate

RWR SHALL represent each supported version of a blueprint type as its own struct
describing exactly what that version accepts on disk. Decoding SHALL select the
struct by resolved version and then convert it to the canonical type the processors
consume.

Nothing downstream of the decoder SHALL need to know that more than one version
exists.

Adding a version SHALL require only: the new wire struct, a conversion to the
canonical type, a registry entry, and the version added to that type's supported
list.

#### Scenario: Adding a version

- **WHEN** a contributor adds a v2 for `packages`
- **THEN** the packages processor still consumes the canonical packages type
  unchanged
- **AND** no other blueprint type is touched

### Requirement: A version declaration is readable even when the rest is not

RWR SHALL read the `schema_version` key independently of the rest of the file, so a
blueprint written in a version this build cannot decode still reports which version
it wanted rather than failing as malformed.

#### Scenario: A future format that will not decode

- **WHEN** a blueprint is written in a wire format this build does not understand
- **AND** it declares `schema_version: 3`
- **THEN** the error reports version 3 as unsupported, rather than a parse failure
