# Install RWR

## Install with a script

On Linux and macOS:

```bash
curl -sSL https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.sh | sudo bash
```

On Windows, in a PowerShell that you started with "Run as administrator":

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force; iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/FynxLabs/rwr/refs/heads/master/install.ps1'))
```

Each script finds the correct build for your machine, compares the download
against the published checksum, and stops if the two do not agree.

WARNING: Read a script before you run it with root or administrator permissions.
Both scripts are in the RWR repository.

## Platforms

RWR publishes a build for each of these:

| Operating system | Architectures |
|---|---|
| Linux | `x86_64`, `arm64`, `armv7`, `riscv64` |
| macOS | `x86_64`, `arm64` |
| Windows | `x86_64`, `arm64` |

These package formats are on the
[releases page](https://github.com/fynxlabs/rwr/releases):

- Archives (`.tar.gz` for Linux and macOS, `.zip` for Windows)
- Debian packages (`.deb`)
- RPM packages (`.rpm`)
- Alpine packages (`.apk`)
- Arch packages (`.pkg.tar.zst`)
- A Homebrew tap

There is no Arch package for `riscv64`, because Arch has no official riscv64
port. On that architecture, use the archive, the `.deb`, the `.rpm`, or the
`.apk`.

## Install from a release

1. Get the archive for your machine from the
   [releases page](https://github.com/fynxlabs/rwr/releases).
2. Compare the file against `checksums.txt` from the same release.
3. Extract the archive.
4. Move the `rwr` binary to a directory in your `PATH`.

## Builds of the master branch

Each merge to `master` publishes a prerelease with the tag
[`nightly`](https://github.com/fynxlabs/rwr/releases/tag/nightly). Use it to test
a correction before the next release.

WARNING: A `nightly` build is not a release. It is the master branch at the time
of the build. The build passed CI, but nobody tested it. Use the
[latest release](https://github.com/fynxlabs/rwr/releases/latest) for a machine
that you depend on.

RWR replaces the `nightly` tag and its files at each merge. The download
addresses stay the same, and the content changes.

The version of a nightly build contains the commit, in the form
`<next-patch>-master-<short-sha>`. To find the commit that made a binary:

```bash
rwr version
```

## Next steps

- [Quick Start](quick-start.md)
- [Configuration File](cli/configuration.md)
