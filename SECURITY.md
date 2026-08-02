# Security Policy

rwr provisions machines: it installs packages, writes system configuration,
manages users and services, and runs elevated commands from blueprint files.
We take security reports seriously — a bug here runs as root on people's
computers.

## Reporting a vulnerability

**Please do not open a public issue for a vulnerability.**

Report it privately via
[GitHub Security Advisories](https://github.com/FynxLabs/rwr/security/advisories/new)
("Report a vulnerability" on the Security tab). You will receive an initial
response in one week or less.

Include what you can: the rwr version (`rwr version`), a minimal blueprint or
provider definition that demonstrates the problem, and what an attacker gains.

## Scope

The reports that are most important to us:

- Anything that lets a blueprint, provider definition, or downloaded content
  do more than the operator asked for — command injection, path traversal
  that escapes the declared boundaries, template injection.
- Local privilege escalation through rwr's elevated operations or staging.
- Credential exposure: tokens or keys reaching logs, process listings,
  subprocess environments, or world-readable files.
- Integrity of the release pipeline and installers (checksums, signatures).

Out of scope: destructive actions that a blueprint declares. Blueprints are
trusted operator input by design. A blueprint can do what the operator can do.
The *data* that a blueprint references (URLs, archives, names) must not widen
this boundary.

## Supported versions

Only the [latest release](https://github.com/FynxLabs/rwr/releases/latest)
receives fixes. The `nightly` prerelease is a rolling build of master and is
not a supported target. Reports against it are welcome. Fixes land there
first.
