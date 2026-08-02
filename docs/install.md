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

Each script finds the correct build for your machine. It downloads
`checksums.txt` from the same release. It compares the SHA-256 of the archive
against the entry for that file. If the two do not agree, or if the release
publishes no checksum for the archive, the script does not install.

`install.sh` installs to `/usr/local/bin`, with `LICENSE` and the README under
`/usr/local/share/doc/rwr`. It uses `sudo` only if those directories are not
writable. If `sudo` is missing, the script reports this before it downloads
anything.
`install.ps1` installs to `%ProgramFiles%\rwr` and adds that directory to the
machine `PATH`, so it needs an elevated PowerShell.

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
`<next-patch>-master-<short-sha>`. To find the commit that made a binary, run
`rwr version`:

```console
$ rwr version
rwr 0.5.3-master-24872aa
commit:     24872aab978e459254544b1fb58afbb080100cb1
built:      2026-08-01T21:20:55Z
built by:   goreleaser
tree state: false
go:         go1.26.5 linux/amd64
```

`rwr --version` prints the first line on its own.

## Next steps

- [Quick Start](quick-start.md)
- [Configuration File](cli/configuration.md)
