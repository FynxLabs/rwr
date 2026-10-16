# Publishing a release

Open **Actions → RWR - Release → Run workflow**, select **master**, choose
**patch**, **minor**, or **major**, and click **Run workflow**.

The workflow tags the commit captured when you start the run, then publishes
through the existing GoReleaser job, including signing, Homebrew, and naur.
Version-tag pushes also continue to start releases.

The bump uses the highest stable `vMAJOR.MINOR.PATCH` tag, ignoring `nightly`
and prereleases. From `v0.6.3`:

| Selection | New tag |
|---|---|
| patch | `v0.6.4` |
| minor | `v0.7.0` |
| major | `v1.0.0` |

With no stable version yet, the base is `v0.0.0`. Major and minor bumps reset
lower components to zero. Release notes use the preceding stable tag.

This follows Reeve's release workflow: inline Bash resolves and pushes the tag,
and the same job publishes it. Stable release runs do not interrupt an active
publish. Each merge also continues to produce the separate nightly prerelease.
