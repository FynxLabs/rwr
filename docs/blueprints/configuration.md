# Configuration Blueprint

With the Configuration Processor in Rinse, Wash, Repeat (RWR), you manage system configurations on Linux (dconf, gsettings), macOS (defaults), and Windows (the registry).

See [Fields Common to Every Blueprint](common-fields.md) for the rule that an
unknown key is an error.

## Blueprint Structure

The Configuration Blueprint has the following structure:

```yaml
configurations:
  - name: string
    tool: string
    elevated: boolean
    run_once: boolean
    profiles:
      - string
```

## Blueprint Settings

The following settings are available for every entry, whichever tool it uses:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes | A unique name for the configuration. It also names the `run_once` marker file |
| `tool` | Yes | The configuration tool: `dconf`, `gsettings`, `macos_defaults` or `windows_registry`. Any other value is an error |
| `profiles` | No | Profiles this entry belongs to. Empty means it is always applied |
| `elevated` | No | Whether to run the configuration with elevated privileges (default: false) |
| `run_once` | No | dconf only: create a marker file and skip the entry on later runs (default: false) |
| `action` | No | Accepted by the schema but **not read**. The `tool` decides what happens. There is nothing to select |
| `names` | No | Accepted by the schema but **not read**. One entry is one operation |

The tool-specific settings are `file`, `schema`, `key`, `settings`, `value`,
`path`, `domain`, `kind` and `type`, described per tool below.

> [!NOTE]
> Profiles work for configuration entries as of this release. Before, the type
> had no `profiles` field, and RWR applied every entry on every machine. The
> configuration blueprint has **no `import` field** and no `interactive` field;
> writing either one is now a decode error.

## Supported Configuration Tools

### dconf (Linux)

The dconf tool loads a dconf dump from a file (`dconf load /`).

| Setting | Required | Description |
|---------|----------|-------------|
| `file` | Yes | Path to the dconf configuration file, resolved relative to the blueprint directory |

`run_once` applies to this tool: with it set, RWR writes a marker file named
`configuration_<name>_bootstrap` into the run-once location and skips the entry
on later runs.

Example:

```yaml
configurations:
  - name: gnome-settings
    tool: dconf
    elevated: true
    run_once: true
    file: ./dconf/settings.ini
```

### gsettings (Linux)

The gsettings tool sets individual keys in one schema.

| Setting | Required | Description |
|---------|----------|-------------|
| `schema` | Yes | The gsettings schema |
| `settings` | Yes | A map of key to value. This is where the keys go — `key` and `value` are **not** read by this tool |

How RWR applies the keys:

- RWR checks each key with `gsettings writable` first.
- A key that is not writable, or that fails to apply, is recorded as a failure.
- RWR reports the failures at the end of the run and still attempts the
  remaining keys.

How RWR formats values:

- Strings are quoted. Lists are written as `[…]`. Booleans are written as
  `true`/`false`.
- A string that starts with `[` or `(` passes through unchanged, so
  pre-formatted values continue to work.

Example:

```yaml
configurations:
  - name: interface-theme
    tool: gsettings
    schema: org.gnome.desktop.interface
    settings:
      gtk-theme: Adwaita-dark
      color-scheme: prefer-dark
      enable-animations: false
```

### macos_defaults (macOS)

The macos_defaults tool runs `defaults write`.

| Setting | Required | Description |
|---------|----------|-------------|
| `domain` | No | The defaults domain. Omit for `NSGlobalDomain` |
| `key` | Yes | The key to set |
| `kind` | Yes | The `defaults` type flag, without the dash: `string`, `bool`, `int`, `float`, … |
| `value` | Yes | The value to set |

Example:

```yaml
configurations:
  - name: dock-orientation
    tool: macos_defaults
    domain: com.apple.dock
    key: orientation
    kind: string
    value: right
```

### windows_registry (Windows)

The windows_registry tool writes a single value under `HKLM:`.

| Setting | Required | Description |
|---------|----------|-------------|
| `path` | Yes | The registry key path. `HKLM:\` is prefixed for you, so give the path from there |
| `key` | Yes | The registry value name |
| `type` | Yes | `string`, `expandstring`, `binary`, `dword` or `qword`. Any other value is an error |
| `value` | Yes | The value to write |

Value shapes:

- `dword` and `qword` values must be whole numbers (a numeric string is
  accepted).
- `binary` takes a list of byte values 0–255.
- `string` and `expandstring` take a string.
- A value of the wrong shape is a named error rather than garbage in the
  registry.

> [!IMPORTANT]
> `elevated: true` raises a UAC prompt, so the run is not unattended: someone has
> to approve it. Run rwr from an already-elevated shell to avoid the prompt.

RWR passes the path, name, and value to PowerShell as environment variables,
not interpolated into the command. PowerShell never parses anything a
blueprint supplies.

Example:

```yaml
configurations:
  - name: disable-uac
    tool: windows_registry
    elevated: true
    path: SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System
    key: EnableLUA
    type: dword
    value: 0
```

## Notes

* Only the dconf tool honors `run_once`. The other three tools apply their setting on every run. Each of them is idempotent.
* The `elevated` setting runs the command through sudo on Unix-like systems. On Windows it does not raise privileges — see the note above.
* A gsettings entry that cannot apply a key does not stop the run. RWR collects the failures and reports them at the end. The other tools return their error immediately.

For more information, see the [Blueprints Overview](../blueprints-general.md) and [Best Practices](../best-practices.md).
