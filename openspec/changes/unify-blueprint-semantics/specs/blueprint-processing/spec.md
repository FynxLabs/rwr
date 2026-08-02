# Blueprint Processing — Deltas

## MODIFIED Requirements

### Requirement: Downloads are validated on every hop, bounded, and pinnable

(Extends the existing requirement with the schema half.)

A repository MAY declare `key_sha256`, the sha256 of the signing key at
`key_url`; when declared, the provider's key-download step SHALL verify it.
A `files:` entry with a URL source MAY declare `sha256`; when declared, the
download SHALL be verified before the file is installed.

A repository with `key_url` and no `key_sha256`, and a files entry with a URL
source and no `sha256`, SHALL produce a prominent warning naming the unpinned
download. (A later major version refuses; the policy ratchets, never loosens.)

#### Scenario: Pinned signing key mismatch

- **WHEN** a repository declares `key_url` and `key_sha256` and the downloaded
  key does not match
- **THEN** the repository is not added and the failure names the digest
  mismatch

#### Scenario: Unpinned signing key warns

- **WHEN** a repository declares `key_url` without `key_sha256`
- **THEN** the run proceeds with a prominent warning naming the repository

### Requirement: Any blueprint entry can import another file

(Modifies base-directory resolution.)

An import path SHALL resolve relative to the file that declares it, in every
processor. (Previously six processors resolved top-level imports against the
tree root while four resolved file-relative — the same `import:` string meant
two different files depending on the blueprint type.)

#### Scenario: Same relative import in two blueprint types

- **GIVEN** `packages/base.yaml` and `files/base.yaml`, each declaring
  `import: ../shared/common.yaml`
- **WHEN** both are processed
- **THEN** both resolve `shared/common.yaml` at the tree root's sibling —
  the same file, relative to each importing file

### Requirement: `names` wins over `name` everywhere

When an entry declares both `name` and `names`, the `names` list SHALL be
processed and `name` ignored, for every blueprint type. (packages had it
backwards relative to files/fonts.)

#### Scenario: Both declared on a packages entry

- **WHEN** a packages entry declares `name: git` and `names: [vim, tmux]`
- **THEN** vim and tmux are installed and git is not

### Requirement: Blueprint type is derived from content when the path names none

A blueprint file whose path names no processor directory SHALL be typed by its
top-level keys (`packages:` → packages, `repositories:` → repositories, …).
A file whose content matches no type — or more than one — SHALL produce the
loud unrouted-file warning and SHALL NOT execute.

Why: the flattened and minimal_files layouts the examples ship were dead ends —
path-only dispatch sent every file to a bucket the run loop never reads, and
the run exited 0 having executed nothing.

#### Scenario: Flattened layout executes

- **GIVEN** a tree with `packages.yaml` at its root declaring packages
- **WHEN** `rwr all` runs
- **THEN** the file is dispatched to the packages processor and executes

#### Scenario: Ambiguous content still warns

- **GIVEN** a root-level file declaring both `packages:` and `services:`
- **WHEN** the run order is computed
- **THEN** the file is not executed and a warning names it and why
