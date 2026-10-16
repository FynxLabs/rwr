# Distribution Specification

## Purpose

RWR is the first thing installed on a new machine, often by piping a script into a
root shell. This capability defines what is published, for which platforms, and what
the install scripts must verify before running anything.

## Requirements

### Requirement: Builds are published for every supported platform

RWR SHALL publish a build for each of:

| Operating system | Architectures |
|---|---|
| Linux | `x86_64`, `arm64`, `armv7`, `riscv64` |
| macOS | `x86_64`, `arm64` |
| Windows | `x86_64`, `arm64` |

RWR SHALL publish archives, Debian, RPM, Alpine, and Arch packages, and a Homebrew
tap.

There SHALL be no Arch package for `riscv64`, because Arch has no official riscv64
port.

#### Scenario: Installing on a riscv64 Linux machine

- **WHEN** an operator installs on riscv64
- **THEN** an archive, `.deb`, `.rpm`, and `.apk` are available and no `.pkg.tar.zst`

### Requirement: Install scripts verify what they download

Each install script SHALL detect the machine's operating system and architecture,
select the matching build, fetch the published `checksums.txt`, and compare the
download against it. A script SHALL abort when the checksums do not agree, when the
checksums file is not published, when it holds no entry for the selected asset, or
when no hashing tool is available to perform the check.

A script SHALL match a published asset by its whole file name, so
`rwr_Linux_arm64.tar.gz` can never satisfy a request for `rwr_Linux_arm.tar.gz`.

A script SHALL accept only architectures goreleaser actually builds - `x86_64`,
`arm64`, and, on Linux only, `armv7` and `riscv64` - and SHALL say so when asked for
anything else. There is deliberately no `i386` case: no 386 target is built, so
accepting it only produced a misleading "could not find a download URL" later on.

`install.sh` SHALL run under `set -euo pipefail`, so an unset variable or a failing
stage of a pipeline aborts rather than continuing with an empty value.

A script SHALL use a per-run temporary directory rather than a fixed path, because a
fixed path in a world-writable directory combined with an elevated move is a local
privilege-escalation route. `install.sh` SHALL create it with `mktemp -d` and remove
it on exit; `install.ps1` SHALL create a GUID-named directory under the temp path.

A script SHALL work out whether it needs `sudo` for the install locations before
downloading anything, so a missing `sudo` is reported up front rather than halfway
through.

#### Scenario: A tampered download

- **WHEN** the downloaded archive does not match the published checksum
- **THEN** the script aborts and installs nothing

#### Scenario: No hashing tool available

- **WHEN** neither `sha256sum` nor `shasum` is present
- **THEN** the script refuses to install rather than skipping verification

#### Scenario: An unbuilt architecture

- **WHEN** the shell installer runs on an i386 machine
- **THEN** it reports the architecture as unsupported and names the ones that are
  published

#### Scenario: Detecting architecture on Windows

- **WHEN** the PowerShell installer runs on an arm64 machine
- **THEN** it selects the arm64 build

### Requirement: Every merge to master publishes a prerelease

Each merge to `master` SHALL publish a prerelease under a fixed `nightly` tag, so a
correction can be tested before the next release. RWR SHALL replace the tag and its
files on each merge, leaving the download addresses stable.

The version SHALL contain the commit, as `<next-patch>-master-<short-sha>`, so a
binary can be traced to the commit that produced it.

A prerelease SHALL be marked as not a release: it is master at the time of the build,
CI-passing but untested.

#### Scenario: Identifying a nightly build

- **WHEN** `rwr version` runs on a nightly build
- **THEN** the version contains the short commit SHA it was built from

### Requirement: CI enforces the security checks

Every pull request SHALL run, as gating checks:

- a build and `go vet` on Linux, macOS, and Windows, because RWR provisions all
  three and building only on Linux meant the Windows and darwin paths were never
  compiled by CI;
- the test suite with `-race` on Linux, matching the pre-push hook;
- `golangci-lint`, reading the repository's own `.golangci.yml` so there is no
  second copy of the configuration to drift;
- `shellcheck` on `install.sh`, and a PowerShell parse plus a gating
  PSScriptAnalyzer run on `install.ps1`, without executing either - running them
  would test the release rather than the script;
- the example blueprint checks, including `rwr validate` over every example tree;
- `gosec` and `govulncheck`.

CI SHALL run on every pull request regardless of target branch, so a branch stacked
on another PR is not merged with no CI at all.

A suppression SHALL be scoped to a specific rule and carry a justification. Blanket
suppressions SHALL NOT be used.

#### Scenario: A change that introduces a vulnerable dependency

- **WHEN** a pull request upgrades to a module with a known advisory
- **THEN** CI fails on the vulnerability check

#### Scenario: A change that breaks the Windows build

- **WHEN** a pull request breaks `internal/system/windows.go`
- **THEN** the Windows leg of the build matrix fails

#### Scenario: A shell error in the installer

- **WHEN** `install.sh` gains a shellcheck error
- **THEN** the installers job fails

### Requirement: Documentation states what is tested

Documentation SHALL describe only commands and fields that exist. Windows support
SHALL be described honestly as incompletely tested rather than as equivalent to Linux
and macOS.

#### Scenario: An operator following the documentation

- **WHEN** an operator runs a command exactly as documented
- **THEN** the command exists and behaves as described


### Requirement: Operators can select a stable release bump in GitHub Actions

The stable release workflow SHALL support `workflow_dispatch` with a required
choice of `major`, `minor`, or `patch`, defaulting to `patch`. An optional full
commit SHA SHALL select a specific commit already merged into master; otherwise
the dispatch's master revision SHALL be used. Manual runs from another branch
SHALL fail before creating a tag.

The version base SHALL be the highest stable `vMAJOR.MINOR.PATCH` tag, ignoring
nightly, prereleases, and non-version tags. Major/minor increments SHALL reset
lower components. The target SHALL include the latest stable release. With no
stable tags the base SHALL be `v0.0.0`.

#### Scenario: A minor release of a specific merge

- **GIVEN** the latest stable version is `v0.6.3`
- **WHEN** an operator chooses minor and supplies a merged commit SHA
- **THEN** the workflow creates `v0.7.0` at that exact commit
- **AND** publishes it through the existing signed GoReleaser pipeline

#### Scenario: A patch release without a commit override

- **WHEN** the operator chooses patch and leaves the commit field blank
- **THEN** the tag points to the master revision captured at dispatch time
- **AND** subsequent master changes do not alter the release's contents

### Requirement: Release reservation is serialized and retryable

Manual and version-tag releases SHALL share a concurrency group that does not
cancel an active publish. Manual version tags SHALL be annotated with their
workflow run ID. Retrying that run SHALL reuse its reserved tag and commit,
without force-pushing a tag or incrementing again. Another manual run for an
already versioned commit SHALL be rejected. A conflicting remote tag SHALL fail
rather than be overwritten.

#### Scenario: Publishing fails after tag creation

- **WHEN** the original workflow run is retried
- **THEN** preparation returns the original tag and target SHA
- **AND** does not allocate a second version

#### Scenario: A non-master commit or stale target is selected

- **WHEN** the selected commit is not merged into master or excludes the latest stable release
- **THEN** preparation fails without creating a release tag

### Requirement: Both release triggers use one publishing path

Manual releases SHALL invoke the publishing job in the same workflow run,
without depending on a push event from the built-in workflow token. Existing
`v*` tag pushes SHALL continue to invoke the publisher, while the master nightly
workflow SHALL remain separate. The publisher SHALL check out the selected tag
and explicitly supply that tag to GoReleaser.

#### Scenario: A tag is created by the manual preparation job

- **WHEN** the built-in workflow token pushes the reserved tag
- **THEN** the same run publishes the release without an extra token
- **AND** the tag push is not required to create another workflow run

## Known Gaps

- **The test suite is not gating on macOS or Windows.** It runs on both, but with
  `continue-on-error`: much of the suite assumes POSIX behaviour (unix file modes,
  `/bin/bash`, path separators), and gating on it immediately would fail master for
  reasons unrelated to the change under review. The failures need triaging at their
  source before the escape hatch comes off - until then the platforms RWR
  provisions are built but not really tested.
- **Neither installer can pass an init file through to a first run.** See the
  initialization specification.
