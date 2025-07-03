# The Fonts Blueprint

The Fonts Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage fonts on your system. You can install, remove, and manage fonts from various providers, with a current focus on Nerd Fonts. This blueprint type simplifies the process of maintaining consistent font configurations across different systems.

## Blueprint Structure

The Fonts Blueprint has the following structure:

```yaml
fonts:
  - name: <font_name>
    action: <action>
    provider: <provider>
    location: <location>
  - names:
      - <font_name1>
      - <font_name2>
    action: <action>
    provider: <provider>
    location: <location>
```

## Blueprint Settings

The following settings are available for the Fonts Blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes* | The name of the font to manage. Use "AllFonts" to manage all available fonts. |
| `names` | Yes* | A list of font names to manage. Used when managing multiple fonts in a single entry. |
| `action` | Yes | The action to perform on the font(s). Valid values are `install` and `remove`. |
| `provider` | No | The font provider to use. Currently, only "nerd" (Nerd Fonts) is supported. Defaults to "nerd" if not specified. |
| `location` | No | Where to install the font. Valid values are `local` (user's home directory) and `system` (system-wide). Defaults to `local` if not specified. |

*Note: Either `name` or `names` must be provided, but not both.

## Font Processing

The Fonts Blueprint manages fonts based on the specified actions:

### Installation

When the `action` is set to `install`, RWR will download and install the specified font(s) using the appropriate provider (currently Nerd Fonts). The installation process differs based on the `location`:

- `local`: Installs the font(s) in the user's home directory.
- `system`: Installs the font(s) system-wide, which requires elevated privileges.

### Removal

When the `action` is set to `remove`, RWR will remove the specified font(s) from the system. The removal process also respects the `location` setting.

## Provider Support

Currently, the Fonts Blueprint supports the following providers:

- Nerd Fonts: A collection of fonts patched with extra glyphs, particularly useful for developers and power users.

## Examples

Here are some examples of using the Fonts Blueprint in YAML, JSON, and TOML formats:

### Installing a Single Font

#### YAML

```yaml
fonts:
  - name: Hack
    action: install
    provider: nerd
    location: local
```

#### JSON

```json
{
  "fonts": [
    {
      "name": "Hack",
      "action": "install",
      "provider": "nerd",
      "location": "local"
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
location = "local"
```

### Installing Multiple Fonts

#### Multiple Fonts YAML

```yaml
fonts:
  - names:
      - Hack
      - SauceCodePro
    action: install
    provider: nerd
    location: system
```

#### Multiple Fonts JSON

```json
{
  "fonts": [
    {
      "names": ["Hack", "SauceCodePro"],
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
names = ["Hack", "SauceCodePro"]
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
    location: local
```

#### Font Removal JSON

```json
{
  "fonts": [
    {
      "name": "Hack",
      "action": "remove",
      "location": "local"
    }
  ]
}
```

#### Font Removal TOML

```toml
[[fonts]]
name = "Hack"
action = "remove"
location = "local"
```

### Installing All Available Fonts

#### All Fonts YAML

```yaml
fonts:
  - name: AllFonts
    action: install
    provider: nerd
    location: system
```

#### All Fonts JSON

```json
{
  "fonts": [
    {
      "name": "AllFonts",
      "action": "install",
      "provider": "nerd",
      "location": "system"
    }
  ]
}
```

#### All Fonts TOML

```toml
[[fonts]]
name = "AllFonts"
action = "install"
provider = "nerd"
location = "system"
```

## Notes

- Installing fonts system-wide (`location: system`) requires elevated privileges.
- The `AllFonts` option for the `name` field will process all available fonts from the specified provider.
- When using the `names` field to specify multiple fonts, all listed fonts will be processed with the same action and settings.
- The Fonts Blueprint uses the Nerd Fonts installation script for managing fonts. Ensure that the necessary dependencies for this script are available on your system.

For more information on using the Fonts Blueprint in your RWR configuration, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) guide.
