# Blueprint Processing - Deltas

## ADDED Requirements

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
A file whose content matches no type - or more than one - SHALL produce the
loud unrouted-file warning and SHALL NOT execute.

Why: the flattened and minimal_files layouts the examples ship were dead ends -
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

## MODIFIED Requirements

### Requirement: Downloads are validated on every hop, bounded, and pinnable

A download URL SHALL be refused unless it is https (plain http is allowed only
for loopback hosts) - everything RWR downloads (package-signing keys, fonts,
file sources, the init file) is installed with the operator's privileges - and
that check SHALL be re-applied to **every redirect hop**, not only the initial
URL. Validating only the first URL is worthless the moment a server answers
with a 302 to plain http.

Downloads SHALL time out rather than hang a run indefinitely.

Where a step or blueprint entry declares a `sha256` for downloaded content, RWR
SHALL verify the download against it before the content is moved into place,
and SHALL discard the download on a mismatch. This applies to repository action
steps and package-manager install/remove steps alike.

A repository MAY declare `key_sha256`, the sha256 of the signing key at
`key_url`; when declared, the provider's key-download step SHALL verify it.
A `files:` entry with a URL source MAY declare `sha256`; when declared, the
download SHALL be verified before the file is installed.

A repository with `key_url` and no `key_sha256`, and a files entry with a URL
source and no `sha256`, SHALL produce a prominent warning naming the unpinned
download. (A later major version refuses; the policy ratchets, never loosens.)

#### Scenario: A redirect to plain http

- **WHEN** an https download URL answers with a redirect to a plain-http,
  non-loopback URL
- **THEN** the download is refused before any request is made to the redirect
  target

#### Scenario: An install step with a wrong digest

- **WHEN** a provider's install step declares `sha256` and the downloaded
  content does not match it
- **THEN** the step fails, the staged download is discarded, and nothing is
  written to the destination

#### Scenario: Pinned signing key mismatch

- **WHEN** a repository declares `key_url` and `key_sha256` and the downloaded
  key does not match
- **THEN** the repository is not added and the failure names the digest
  mismatch

#### Scenario: Unpinned signing key warns

- **WHEN** a repository declares `key_url` without `key_sha256`
- **THEN** the run proceeds with a prominent warning naming the repository

### Requirement: Any blueprint entry can import another file

RWR SHALL accept an `import` field on a blueprint entry, naming another blueprint
file whose definitions are merged in.

Import paths SHALL resolve relative to the file that declares them, in every
processor. (Previously six processors resolved top-level imports against the
tree root while four resolved file-relative - the same `import:` string meant
two different files depending on the blueprint type.) An imported file MAY be
in any supported format, and its format SHALL be derived from its own
extension, not the importing file's. Imported entries SHALL be subject to
profile filtering like any other entry. Import SHALL work for every blueprint
type.

RWR SHALL detect a circular import and SHALL NOT loop.

#### Scenario: Sharing a common package set

- **WHEN** a machine-specific blueprint imports `../../Common/packages/base.yaml`
- **AND** adds its own entries
- **THEN** both the imported and the local entries are applied

#### Scenario: A circular import

- **WHEN** file A imports B and B imports A
- **THEN** the cycle is detected and the run does not loop

#### Scenario: A nested import chain

- **WHEN** A imports B and B imports C
- **THEN** the definitions from all three files are applied

#### Scenario: Same relative import in two blueprint types

- **GIVEN** `packages/base.yaml` and `files/base.yaml`, each declaring
  `import: ../shared/common.yaml`
- **WHEN** both are processed
- **THEN** both resolve the same file, relative to each importing file
