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

## Known Gaps

- **The test suite is not gating on macOS or Windows.** It runs on both, but with
  `continue-on-error`: much of the suite assumes POSIX behaviour (unix file modes,
  `/bin/bash`, path separators), and gating on it immediately would fail master for
  reasons unrelated to the change under review. The failures need triaging at their
  source before the escape hatch comes off - until then the platforms RWR
  provisions are built but not really tested.
- **Neither installer can pass an init file through to a first run.** See the
  initialization specification.
