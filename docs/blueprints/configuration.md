# Configuration Blueprint

The Configuration Processor manages system and desktop settings across operating
systems. Its tools are dconf and gsettings for Linux, defaults for macOS, the
Windows Registry, and post-login Omarchy desktop configuration.

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
| `name` | Yes for resources | A unique name for the configuration. Omit it on an Omarchy import entry. It also names the `run_once` marker file |
| `tool` | Yes | The configuration tool: `dconf`, `gsettings`, `macos_defaults`, `windows_registry`, or `omarchy`. Any other value is an error |
| `profiles` | No | Profiles this entry belongs to. Empty means it is always applied |
| `elevated` | No | Whether to run a platform configuration command with elevated privileges (default: false). Omarchy entries must omit it |
| `run_once` | No | dconf only: create a marker file and skip the entry on later runs (default: false) |
| `action` | No | `set` is the only supported action; leaving it out means the same thing. Omit it on an import entry |
| `names` | No | Accepted for existing platform tools but **not read**. One entry is one operation. Omarchy entries reject it |

The tool-specific settings are described with each tool below. Omarchy entries
reject fields belonging to the other tools, and the other tools reject Omarchy
resource fields.

> [!NOTE]
> Profiles work for every configuration tool. Omarchy entries also support
> `import`; the other tools reject it. Configuration entries do not support
> `interactive`.

## Supported Configuration Tools

### omarchy (Linux)

The Omarchy tool reconciles desktop resources after login. It uses the same
`configurations` list and the same `configuration` processor as every tool on
this page:

```yaml
configurations:
  - name: omarchy-desktop
    tool: omarchy
    action: set
    plugins:
      - id: expose.window-overview
        source:
          git: https://github.com/kristofferR/omarchy-expose.git
        enabled: true
        settings:
          hotCornerEnabled: false
    defaults:
      browser: firefox
      terminal: ghostty
      editor: nvim
```

Run it with `rwr run configuration`, or let `rwr all` apply it with the other
configuration entries. Omarchy configuration always runs as the logged-in
desktop user and does not accept `elevated` or `run_once`.

Omarchy entries can manage plugin installation and activation, targeted plugin
and shell settings, theme selection, default applications, hooks, widgets, and
an external screensaver launcher. They support nested, cross-format imports;
paths resolve relative to the file that declares them. See the
[Omarchy configuration tool](omarchy.md) for the complete schema and runtime
requirements.

### dconf (Linux)

The dconf tool loads a dconf dump from a file (`dconf load /`).

| Option | Required | Description |
|--------|----------|-------------|
| `file` | Yes | Path to the dconf configuration file, resolved relative to the blueprint directory |

`run_once` applies to this tool: with it set, RWR writes a marker file named
`configuration_<name>_bootstrap` into the run-once location and skips the entry
on later runs. The marker is written only after dconf applies successfully. If
the marker cannot be written, the configuration processor returns an error so
the run cannot report a durable run-once result that was never recorded.

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

| Option | Required | Description |
|--------|----------|-------------|
| `schema` | Yes | The gsettings schema |
| `settings` | Yes | A map of key to value. This is where the keys go - `key` and `value` are **not** read by this tool |

Each key is checked with `gsettings writable` first. A key that is not writable,
or that fails to apply, is recorded as a failure and reported at the end of the
run; the remaining keys are still attempted.

Values are formatted for gsettings automatically: a string is quoted, a list
becomes `[…]`, booleans become `true`/`false`. A string that already starts with
`[` or `(` is passed through unchanged, so pre-formatted values keep working.

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

| Option | Required | Description |
|--------|----------|-------------|
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

| Option | Required | Description |
|--------|----------|-------------|
| `path` | Yes | The registry key path. `HKLM:\` is prefixed for you, so give the path from there |
| `key` | Yes | The registry value name |
| `type` | Yes | `string`, `expandstring`, `binary`, `dword` or `qword`. Any other value is an error |
| `value` | Yes | The value to write |

`dword` and `qword` values must be whole numbers (a numeric string is accepted);
`binary` takes a list of byte values 0–255; `string` and `expandstring` take a
string. A value of the wrong shape is a named error rather than garbage in the
registry.

> [!IMPORTANT]
> `elevated: true` raises a UAC prompt, so the run is not unattended: someone has
> to approve it. Run rwr from an already-elevated shell to avoid the prompt.

The path, name and value are passed to PowerShell as environment variables, not
interpolated into the command, so nothing a blueprint supplies is parsed as
PowerShell.

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

* `run_once` is honoured by the dconf tool only. The other tools reconcile their settings on every run.
* The `elevated` option runs the command through sudo on Unix-like systems. On Windows it does not raise privileges - see the note above.
* A gsettings entry that cannot apply a key does not stop the run; the failures are collected and reported at the end.

For more information on using the Configuration Processor in your RWR setup, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) sections of the documentation.
