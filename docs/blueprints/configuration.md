# Configuration Blueprint

The Configuration Processor in Rinse, Wash, Repeat (RWR) allows you to manage system configurations across different operating systems. It supports various configuration tools including dconf and gsettings for Linux, defaults for macOS, and registry settings for Windows.

## Blueprint Structure

The Configuration Blueprint has the following structure:

```yaml
configurations:
  - name: string
    tool: string
    action: string
    elevated: boolean
    run_once: boolean
```

## Blueprint Settings

The following settings are available for the Configuration Blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes | A unique name for the configuration |
| `tool` | Yes | The configuration tool to use (e.g., "dconf", "gsettings", "macos_defaults", "windows_registry") |
| `action` | Yes | The action to perform (e.g., "set", "load") |
| `elevated` | No | Whether to run the configuration with elevated privileges (default: false) |
| `run_once` | No | Whether to run the configuration only once (default: false) |

## Supported Configuration Tools

### dconf (Linux)

The dconf tool allows loading configurations from a file.

| Option | Required | Description |
|--------|----------|-------------|
| `file` | Yes | Path to the dconf configuration file to load |

Example:

```yaml
configurations:
  - name: Load GNOME settings
    tool: dconf
    action: load
    elevated: true
    run_once: true
    file: /path/to/dconf/settings.ini
```

### gsettings (Linux)

The gsettings tool allows setting individual configuration values.

| Option | Required | Description |
|--------|----------|-------------|
| `schema` | Yes | The gsettings schema |
| `path` | No | The gsettings path (if applicable) |
| `key` | Yes | The key to set |
| `value` | Yes | The value to set |

Example:

```yaml
configurations:
  - name: Set GNOME theme
    tool: gsettings
    action: set
    schema: org.gnome.desktop.interface
    key: gtk-theme
    value: "'Adwaita-dark'"
```

### macos_defaults (macOS)

The macos_defaults tool allows setting macOS system defaults.

| Option | Required | Description |
|--------|----------|-------------|
| `domain` | No | The defaults domain (omit for global defaults) |
| `key` | Yes | The key to set |
| `kind` | Yes | The type of the value (e.g., "string", "bool", "int") |
| `value` | Yes | The value to set |

Example:

```yaml
configurations:
  - name: Set dock orientation
    tool: macos_defaults
    action: set
    domain: com.apple.dock
    key: orientation
    kind: string
    value: right
```

### windows_registry (Windows)

The windows_registry tool allows setting Windows registry values.

| Option | Required | Description |
|--------|----------|-------------|
| `path` | Yes | The registry key path |
| `key` | Yes | The registry value name |
| `type` | Yes | The type of the value (e.g., "string", "dword", "qword") |
| `value` | Yes | The value to set |

Example:

```yaml
configurations:
  - name: Disable UAC
    tool: windows_registry
    action: set
    elevated: true
    path: SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System
    key: EnableLUA
    type: dword
    value: 0
```

## Notes

* The `run_once` option, when set to true, will create a marker file to ensure the configuration is only applied once. This is useful for one-time system configurations.
* The `elevated` option, when set to true, will attempt to run the configuration with elevated privileges. This may require sudo on Unix-like systems or UAC on Windows.
* For Windows registry operations, the processor uses PowerShell commands to modify the registry, allowing for cross-platform compatibility of the RWR codebase.

For more information on using the Configuration Processor in your RWR setup, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) sections of the documentation.
