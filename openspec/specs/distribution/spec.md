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
select the matching build, fetch the published checksums, and compare the download
against them. A script SHALL abort when the checksums do not agree.

A script SHALL use a per-run temporary directory rather than a fixed path, because a
fixed path in a world-writable directory combined with an elevated move is a local
privilege-escalation route.

#### Scenario: A tampered download

- **WHEN** the downloaded archive does not match the published checksum
- **THEN** the script aborts and installs nothing

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

Every pull request SHALL run the build, the tests, the linter, a static security
scan, and a vulnerability check against the module dependencies.

A suppression SHALL be scoped to a specific rule and carry a justification. Blanket
suppressions SHALL NOT be used.

#### Scenario: A change that introduces a vulnerable dependency

- **WHEN** a pull request upgrades to a module with a known advisory
- **THEN** CI fails on the vulnerability check

### Requirement: Documentation states what is tested

Documentation SHALL describe only commands and fields that exist. Windows support
SHALL be described honestly as incompletely tested rather than as equivalent to Linux
and macOS.

#### Scenario: An operator following the documentation

- **WHEN** an operator runs a command exactly as documented
- **THEN** the command exists and behaves as described
