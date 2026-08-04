# Repositories Blueprint

The Repositories Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage repositories for various package managers. You can add or remove repositories based on your system's requirements.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

```yaml
repositories:
  - name: string
    package_manager: string
    action: string
    url: string
```

## Blueprint Settings

Which of the optional settings matter depends on the package manager: each
provider declares the steps it runs to add or remove a repository, and those
steps are Go templates rendered against the values below. A value a provider
never references is simply unused; a placeholder a provider references and the
blueprint did not supply is an error naming the repository, not a blank.

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the repository. Also names the file most providers write (`/etc/apt/sources.list.d/<name>.list`) and the keyring (`<keys_dir>/<name>.gpg`) |
| `package_manager` | Yes | The provider that owns the repository (`apt`, `dnf`, `pacman`, `zypper`, `apk`, `xbps`, `eopkg`, `emerge`, `slackpkg`, `brew`, `macports`, `nix`, `flatpak`, `snap`, `cargo`, `chocolatey`, `scoop`, `winget`, `mas`, `gnome-extensions`, and the AUR helpers) |
| `action` | Yes | `add` or `remove` |
| `url` | Yes for `add` | The repository URL. A value with no scheme, or a `file://` URL, is treated as a local path - that is what selects the sideload steps for `snap` and `gnome-extensions` |
| `key_url` | No | URL of the repository's signing key, downloaded to the provider's key directory (apt, dnf, zypper, apk, xbps, eopkg, slackpkg) |
| `key_sha256` | No | SHA-256 digest of the signing key at `key_url`; the key download is verified against it |
| `key_id` | No | Key ID to import or delete rather than a URL (pacman family, and the `rpm --erase gpg-pubkey-<id>` step in dnf/zypper removal) |
| `arch` | No | Architecture written into an apt source line |
| `channel` | No | Suite/channel written into an apt source line |
| `component` | No | Component written into an apt source line |
| `repository` | No | Free-form repository name available to provider steps |
| `description` | No | Human-readable description; becomes `name=` in a dnf `.repo` file |
| `overlay_path` | No | Location of a Portage overlay (`emerge`), and what marks a nix entry as an overlay |
| `sync_type` | No | Portage `sync-type`, e.g. `git` or `rsync` |
| `sha256` | No | Digest of a fetched nix overlay tarball |
| `proxy_url` | No | Proxy snapd should route through; setting it adds the `snap set system proxy.*` steps |
| `uuid` | No | GNOME extension UUID, used to enable, disable and uninstall the extension |
| `extension_id` | No | GNOME extension ID, used when installing from extensions.gnome.org |
| `interface` | No | Snap interface to connect after install (`snap connect <name>:<interface> <slot>`) |
| `slot` | No | The slot the snap interface plugs into |
| `reset_settings` | No | GNOME extensions only: also reset the extension's settings when removing it (default `false`) |
| `username` | No | Username for a private source (chocolatey). See the warning below |
| `password` | No | Password for a private source (chocolatey). See the warning below |
| `token` | No | Registry token (`cargo login`). See the warning below |
| `profiles` | No | See [common fields](common-fields.md) |
| `import` | No | See [common fields](common-fields.md) |
| `interactive` | No | See [common fields](common-fields.md) |

These fields used to be parsed and then dropped: only a bare `{{ .URL }}`
argument was ever substituted, so an apt add wrote a file literally named
`{{ .SourcesPath }}/{{ .Name }}.list`. Every field of a provider step is now
rendered, so the settings above take effect.

> [!WARNING]
> `username`, `password` and `token` are passed to the package manager as
> **command-line arguments** - `choco source add --password=…`, `cargo login …`
> - because those tools accept them no other way. They are therefore visible in
> `ps` to every local user on the machine for the lifetime of the call. RWR
> keeps them out of its own output (logs, `--debug` argv dumps and `--dry-run`
> lines redact them), but nothing rwr can do hides them from the process list.

## Actions and what actually works

Every shipped provider - apt, dnf, zypper, pacman/yay/paru/aura/pamac/trizen,
apk, xbps, eopkg, emerge, slackpkg, brew, macports, nix, flatpak, snap, cargo,
chocolatey, scoop, winget, mas, gnome-extensions - defines both `add` and
`remove` steps, and both work. Provider steps may be gated on predicates RWR
derives from the entry (`HasKey`, `HasInterfaces`, `ResetSettings`, …), so a
step for a feature you did not ask for is simply skipped: a snap add without
`interface` does not run `snap connect`, and a GNOME extension remove resets
the extension's settings only when `reset_settings: true`.

Two more things worth knowing:

- A `remove` step that deletes a file is confined to the directories the
  provider declares as its repository paths. A path outside them is refused, and
  symlinked components are resolved before the check.
- After the repository entries are processed, RWR runs the update command of
  every available provider (`apt update`, `pacman -Sy`, …). A failing update is
  a warning, not an error.

## Blueprint Imports

Import repository definitions from other files:

```yaml
repositories:
  # Import shared repositories
  - import: ../../Common/repositories/base-repos.yaml

  # Add environment-specific repositories
  - name: custom-repo
    package_manager: apt
    action: add
    url: https://custom.example.com/repo
    key_url: https://custom.example.com/gpg
    profiles:
      - production
```

This allows you to maintain common repository configurations separately from environment-specific ones.

## Examples

### YAML

```yaml
repositories:
  - name: example-repo
    package_manager: apt
    action: add
    url: https://example.com/repo
    key_url: https://example.com/repo/gpg
    arch: amd64
    channel: stable
    component: main

  - name: epel
    package_manager: dnf
    action: add
    url: https://example.com/epel/$basearch
    key_url: https://example.com/RPM-GPG-KEY-EPEL
    description: Extra Packages for Enterprise Linux

  - name: archlinuxcn
    package_manager: pacman
    action: add
    url: https://repo.archlinuxcn.org/$arch
    key_id: "FBA220DFC880C036"

  - name: another-repo
    package_manager: brew
    action: add
    url: homebrew/cask-fonts
```

### JSON

```json
{
  "repositories": [
    {
      "name": "example-repo",
      "package_manager": "apt",
      "action": "add",
      "url": "https://example.com/repo",
      "key_url": "https://example.com/repo/gpg",
      "arch": "amd64",
      "channel": "stable",
      "component": "main"
    },
    {
      "name": "another-repo",
      "package_manager": "brew",
      "action": "add",
      "url": "homebrew/cask-fonts"
    }
  ]
}
```

### TOML

```toml
[[repositories]]
name = "example-repo"
package_manager = "apt"
action = "add"
url = "https://example.com/repo"
key_url = "https://example.com/repo/gpg"
arch = "amd64"
channel = "stable"
component = "main"

[[repositories]]
name = "another-repo"
package_manager = "brew"
action = "add"
url = "homebrew/cask-fonts"
```

## Notes

- The Repositories Blueprint is processed by the `rwr run repository` command.
- The available package managers and their specific settings vary by operating system. See [Providers](../providers.md) for what each one declares.
- Removing a repository will not automatically remove the packages installed from that repository. You may need to manually remove them using the [Packages Blueprint](packages.md).
