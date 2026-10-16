# Publishing a release

The **RWR - Release** workflow supports both manual releases and existing
`v*` tag pushes. Both paths build the same artifacts, sign checksums, publish
the GitHub release, update the Homebrew tap, and notify naur when its dispatch
secret is configured.

## Release from GitHub Actions

After this workflow is on `master`, open **Actions → RWR - Release → Run
workflow** and select the **master** branch.

1. Select **patch**, **minor**, or **major** under **Release type**.
2. Leave **Merged commit SHA** blank to release the master revision captured
   when you click Run workflow. To release a particular merge instead, paste
   its full 40-character commit SHA. It must already belong to master and
   include the latest stable release.
3. Click **Run workflow**. The preparation job calculates and pushes the new
   annotated version tag. The publishing job checks out that exact tag and
   runs the existing GoReleaser pipeline in the same workflow run.

The highest stable `vMAJOR.MINOR.PATCH` tag is the version base. The `nightly`
tag, prerelease tags, and non-version tags do not affect the calculation.
For example, from `v0.6.3`:

| Selection | New tag |
|---|---|
| patch | `v0.6.4` |
| minor | `v0.7.0` |
| major | `v1.0.0` |

With no stable tags yet, the base is `v0.0.0`. Major and minor releases reset
lower version components to zero. The run summary records the version and
exact commit selected.

## Retry a failed release

Use **Re-run failed jobs** on the existing run. If preparation must run again,
it recognizes its original run ID in the tag annotation and reuses the same
version. It never force-pushes or moves an existing tag. Starting another manual
run for an already versioned commit is rejected so a double click cannot
silently create another version.

Stable release runs share one concurrency group, and an active release is
never automatically cancelled by a newer run. Manual releases publish directly
in their originating run: pushing a tag with GitHub's built-in workflow token
does not start a second tag-push workflow. No additional release token is needed.

## Release by tag

Existing version-tag releases continue to work:

```sh
git tag -a v0.6.4 <commit-sha> -m 'Release v0.6.4'
git push origin v0.6.4
```

Each merge to master also continues to publish the separate `nightly`
prerelease. It does not consume a stable version.
