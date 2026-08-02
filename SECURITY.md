# Security Policy

rwr provisions machines: it installs packages, writes system configuration,
manages users and services, and runs elevated commands from blueprint files.
Security reports are taken seriously — a bug here runs as root on people's
computers.

## Reporting a vulnerability

**Please do not open a public issue for a security problem.**

Report it privately via
[GitHub Security Advisories](https://github.com/FynxLabs/rwr/security/advisories/new)
("Report a vulnerability" on the Security tab). You should receive an initial
response within a week.

Include what you can: the rwr version (`rwr version`), a minimal blueprint or
provider definition that demonstrates the problem, and what an attacker gains.

## Scope

Reports we especially care about:

- Anything that lets a blueprint, provider definition, or downloaded content
  do more than the operator asked for — command injection, path traversal
  escaping declared boundaries, template injection.
- Local privilege escalation through rwr's elevated operations or staging.
- Credential exposure: tokens or keys reaching logs, process listings,
  subprocess environments, or world-readable files.
- Integrity of the release pipeline and installers (checksums, signatures).

Out of scope: blueprints doing destructive things they explicitly declare —
blueprints are trusted operator input by design. The trust boundary is that a
blueprint may do what the operator could do; it must not be possible for
*data* a blueprint references (URLs, archives, names) to widen that.

## Supported versions

Only the [latest release](https://github.com/FynxLabs/rwr/releases/latest)
receives fixes. The `nightly` prerelease is a rolling build of master and is
not a supported target, though reports against it are welcome — that's where
fixes land first.
