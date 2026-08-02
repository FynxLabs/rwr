# The Fonts Blueprint

The Fonts Blueprint in Rinse, Wash, Repeat (RWR) installs and removes Nerd Fonts. Nerd Fonts is the only provider that exists. Each entry names a font in the [nerd-fonts](https://github.com/ryanoasis/nerd-fonts) releases.

See [Fields Common to Every Blueprint](common-fields.md) for the rule that an
unknown key is an error.

## Blueprint Structure

The Fonts Blueprint has the following structure:

```yaml
fonts:
  - name: <font_name>
    action: <action>
    provider: <provider>
    location: <location>
    profiles:
      - <profile>
  - names:
      - <font_name1>
      - <font_name2>
    action: <action>
    location: <location>
```

## Blueprint Settings

The following settings are available for the Fonts Blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes* | The name of the font to manage, exactly as the nerd-fonts release names its archive — `Hack`, `FiraCode`, `JetBrainsMono`. It must not contain a path separator or `..` |
| `names` | Yes* | A list of font names to manage. The rest of the entry is repeated for each |
| `action` | Yes | `install` or `remove` |
| `provider` | No | Defaults to `nerd`. Nerd Fonts is the only implementation. The schema accepts another value, but the value changes nothing |
| `location` | No | `system` installs system-wide. **Any other value, and the default, installs for the current user** |
| `profiles` | No | Profiles this font belongs to. Empty means it is always processed |

*Note: Either `name` or `names` must be provided. An entry with neither is
skipped.

> [!NOTE]
> Profiles work for fonts as of this release. Before, the type had no `profiles`
> field, and a font entry ran on every machine.
> The fonts blueprint has **no `import` field** and no `interactive` field;
> writing either one is now a decode error.

## Font Processing

### Installation

RWR asks GitHub for the latest nerd-fonts release, downloads `<name>.tar.xz`
from it, and extracts the `.ttf` members into the font directory. RWR skips
symlink and hard-link members of the archive. A member whose path goes out of
the font directory stops the install. RWR then refreshes the font cache with
`fc-cache -f -v`.

A name that does not match an archive in the release is a download failure, not
a silent no-op.

### Removal

RWR deletes everything matching `<name>*.ttf` in the font directory and
refreshes the font cache.

### Where fonts go

| `location` | Directory |
|------------|-----------|
| `system`, Linux | `/usr/local/share/fonts` |
| `system`, macOS | `/Library/Fonts` |
| `system`, Windows | `%WINDIR%\Fonts` |
| anything else | `$HOME/.local/share/fonts`, on every platform |

A `system` install writes and refreshes the cache with elevation. A per-user
install does not.

### Dry runs

`--dry-run` lists the fonts to install and makes no network call. Thus it works
offline.

## Examples

Examples in YAML, JSON, and TOML:

### Installing a Single Font

#### YAML

```yaml
fonts:
  - name: Hack
    action: install
    provider: nerd
    location: user
```

#### JSON

```json
{
  "fonts": [
    {
      "name": "Hack",
      "action": "install",
      "provider": "nerd",
      "location": "user"
    }
  ]
}
```

#### TOML

```toml
[[fonts]]
name = "Hack"
action = "install"
provider = "nerd"
location = "user"
```

### Installing Multiple Fonts

#### Multiple Fonts YAML

```yaml
fonts:
  - names:
      - Hack
      - SourceCodePro
    action: install
    provider: nerd
    location: system
```

#### Multiple Fonts JSON

```json
{
  "fonts": [
    {
      "names": ["Hack", "SourceCodePro"],
      "action": "install",
      "provider": "nerd",
      "location": "system"
    }
  ]
}
```

#### Multiple Fonts TOML

```toml
[[fonts]]
names = ["Hack", "SourceCodePro"]
action = "install"
provider = "nerd"
location = "system"
```

### Removing a Font

#### Font Removal YAML

```yaml
fonts:
  - name: Hack
    action: remove
    location: user
```

#### Font Removal JSON

```json
{
  "fonts": [
    {
      "name": "Hack",
      "action": "remove",
      "location": "user"
    }
  ]
}
```

#### Font Removal TOML

```toml
[[fonts]]
name = "Hack"
action = "remove"
location = "user"
```

## Notes

- A system-wide install (`location: system`) requires elevated privileges.
- There is no way to install every font at once. An entry names one archive. Use `names` to list several.
- When you use the `names` field, RWR processes all listed fonts with the same action and settings.
- Installation requires network access to the GitHub API and to the nerd-fonts release assets. `fc-cache` must be on PATH for the cache refresh. A missing `fc-cache` is a warning, not a failure.

For more information, see the [Blueprints Overview](../blueprints-general.md) and [Best Practices](../best-practices.md).
