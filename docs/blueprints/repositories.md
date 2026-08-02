# Repositories Blueprint

With the Repositories Blueprint in Rinse, Wash, Repeat (RWR), you manage repositories for various package managers. You can add or remove repositories based on your system's requirements.

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

Which of the optional settings matter depends on the provider:

- Each provider declares the steps it runs to add or remove a repository.
- Those steps are Go templates that RWR renders against the values below.
- A value that a provider never references is unused.
- A placeholder that a provider references, and that the blueprint did not
  supply, is an error that names the repository, not a blank.

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the repository. Also names the file most providers write (`/etc/apt/sources.list.d/<name>.list`) and the keyring (`<keys_dir>/<name>.gpg`) |
| `package_manager` | Yes | The provider that owns the repository (`apt`, `dnf`, `pacman`, `zypper`, `apk`, `xbps`, `eopkg`, `emerge`, `slackpkg`, `brew`, `macports`, `nix`, `flatpak`, `snap`, `cargo`, `chocolatey`, `scoop`, `winget`, `mas`, `gnome-extensions`, and the AUR helpers) |
| `action` | Yes | `add` or `remove` |
| `url` | Yes for `add` | The repository URL. RWR treats a value with no scheme, or a `file://` URL, as a local path — that is what selects the sideload steps for `snap` and `gnome-extensions` |
| `key_url` | No | URL of the repository's signing key, downloaded to the provider's key directory (apt, dnf, zypper, apk, xbps, eopkg, slackpkg) |
| `key_id` | No | Key ID to import or delete rather than a URL (pacman family, and the `rpm --erase gpg-pubkey-<id>` step in dnf/zypper removal) |
| `arch` | No | Architecture written into an apt source line |
| `channel` | No | Suite/channel written into an apt source line |
| `component` | No | Component written into an apt source line |
| `repository` | No | Free-form repository name available to provider steps |
| `description` | No | Human-readable description. Becomes `name=` in a dnf `.repo` file |
| `overlay_path` | No | Location of a Portage overlay (`emerge`), and what marks a nix entry as an overlay |
| `sync_type` | No | Portage `sync-type`, for example `git` or `rsync` |
| `sha256` | No | Digest of a fetched nix overlay tarball |
| `proxy_url` | No | Proxy that snapd routes through. Setting it adds the `snap set system proxy.*` steps |
| `uuid` | No | GNOME extension UUID, used to enable, disable and uninstall the extension |
| `extension_id` | No | GNOME extension ID, used when installing from extensions.gnome.org |
| `username` | No | Username for a private source (chocolatey). See the warning below |
| `password` | No | Password for a private source (chocolatey). See the warning below |
| `token` | No | Registry token (`cargo login`). See the warning below |
| `profiles` | No | See [common fields](common-fields.md) |
| `import` | No | See [common fields](common-fields.md) |
| `interactive` | No | See [common fields](common-fields.md) |

Earlier versions parsed these fields and then dropped them:

- Only a bare `{{ .URL }}` argument was substituted.
- An apt add wrote a file literally named
  `{{ .SourcesPath }}/{{ .Name }}.list`.

RWR now renders every field of a provider step, so the settings above take
effect.

> [!WARNING]
> RWR passes `username`, `password` and `token` to the package manager as
> **command-line arguments** — `choco source add --password=…`, `cargo login …`
> — because those tools accept them no other way.
>
> - They are visible in `ps` to every local user on the machine for the
>   lifetime of the call.
> - RWR keeps them out of its own output: logs, `--debug` argv dumps, and
>   `--dry-run` lines redact them.
> - But nothing rwr can do hides them from the process list.

## Actions and what actually works

Every shipped provider defines both `add` and `remove` steps. Two of them do not
work, and are documented here rather than left to fail on your machine:

| Provider | State |
|----------|-------|
| `snap`, `action: add` | **Broken.** The last add step of the provider is gated on a `HasInterfaces` predicate that RWR does not derive. An unknown predicate is an error, not a silent skip. Thus every snap repository add fails. Snap removal works |
| `gnome-extensions`, `action: remove` | **Broken.** The final step is gated on a `ResetSettings` predicate that RWR does not derive. Thus removal fails after the disable and uninstall steps run. Installing works |

Everything else — apt, dnf, zypper, pacman/yay/paru/aura/pamac/trizen, apk,
xbps, eopkg, emerge, slackpkg, brew, macports, nix, flatpak, cargo, chocolatey,
scoop, winget, mas — supports both `add` and `remove`.

Note these two points:

- A `remove` step that deletes a file is confined to the directories the
  provider declares as its repository paths. RWR refuses a path outside them,
  and it resolves symlinked components before the check.
- After RWR processes the repository entries, it runs the update command of
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

You can then keep common repository configurations separate from environment-specific ones.

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

- The `rwr run repository` command processes the Repositories Blueprint.
- The available providers and their specific settings vary by operating system. See [Providers](../providers.md) for what each one declares.
- A repository removal does not remove the packages installed from that repository. If necessary, remove them with the [Packages Blueprint](packages.md).
