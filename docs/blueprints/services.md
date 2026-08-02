# Services Blueprint

With the Services Blueprint in Rinse, Wash, Repeat (RWR), you start, stop, enable, and disable services on different operating systems.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

A Services Blueprint is an array of service objects in a YAML, JSON, or TOML file. Each object represents one system service.

```yaml
services:
  - name: nginx
    action: start
    elevated: true
  - name: mysql
    action: stop
    elevated: true
```

## Blueprint Settings

The following settings are available for each service object:

| Setting | Required | Description |
|----------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the service |
| `import` | Yes, if `name` is not provided | Path to import service definitions from another file (relative to blueprint directory) |
| `profiles` | No | List of profiles this service belongs to. If empty, service is always managed (base item) |
| `action` | Yes | The action to perform on the service (start, stop, enable, disable, restart, reload, status, create, delete) |
| `elevated` | No | Whether the service requires elevated privileges (default: false) |
| `target` | No | The target file for the service (used with create and delete actions) |
| `content` | No | The content of the service file (used with the create action) |
| `source` | No | The source file for the service (used with the create action) |
| `file` | No | The file associated with the service (used with the delete action) |
| `interactive` | No | Override global interactive mode for this service (`true`/`false`). If omitted, uses the global `--interactive` flag |

## Blueprint Imports

Import service definitions from other files to share common service configurations:

```yaml
services:
  # Import shared base services
  - import: ../../Common/services/base-services.yaml

  # Add environment-specific services
  - name: custom-app
    action: enable
    elevated: true
    profiles:
      - production
```

You can then keep common service configurations separate from system-specific ones.

## Supported Actions

The Services Blueprint supports the following actions:

- `start`: Start the service
- `stop`: Stop the service
- `enable`: Enable the service to start automatically on system boot
- `disable`: Disable the service from starting automatically on system boot
- `restart`: Restart the service
- `reload`: Reload the service configuration
- `status`: Check the status of the service
- `create`: Create a new service file
- `delete`: Delete an existing service file

> [!NOTE]
> `rwr validate` currently accepts only `enable`, `disable`, `start`, `stop` and
> `restart`, and reports the other four as invalid. The processor runs all nine.
> If you use `reload`, `status`, `create` or `delete`, expect a validation error
> you can ignore.

## Platform-Specific Considerations

The Services Blueprint manages services differently on each operating system:

### Linux (systemd)

On Linux systems with systemd, the Services Blueprint uses the `systemctl` command to manage services. The `create` and `delete` actions manage service unit files in the appropriate systemd directories.

### macOS (launchd)

On macOS, the Services Blueprint uses the `launchctl` command to manage services. The `create` and `delete` actions manage service plist files in the `/Library/LaunchDaemons` directory.

### Windows

On Windows, the Services Blueprint uses the `sc` command to manage services. The `create` and `delete` actions manage service configuration and binaries.

## Examples

Examples in YAML, JSON, and TOML:

### YAML

```yaml
services:
  # Base services - always managed (no profiles field)
  - name: nginx
    action: start
    elevated: true

  - name: ssh
    action: enable
    elevated: true

  # Development profile services
  - name: docker
    profiles:
      - dev
    action: start
    elevated: true

  - name: mysql
    profiles:
      - dev
    action: enable
    elevated: true

  # Production profile services
  - name: postgresql
    profiles:
      - production
    action: start
    elevated: true

  - name: redis
    profiles:
      - production
    action: enable
    elevated: true
```

### JSON

```json
{
  "services": [
    {
      "name": "nginx",
      "action": "start",
      "elevated": true
    },
    {
      "name": "ssh",
      "action": "enable",
      "elevated": true
    },
    {
      "name": "docker",
      "profiles": ["dev"],
      "action": "start",
      "elevated": true
    },
    {
      "name": "mysql",
      "profiles": ["dev"],
      "action": "enable",
      "elevated": true
    },
    {
      "name": "postgresql",
      "profiles": ["production"],
      "action": "start",
      "elevated": true
    },
    {
      "name": "redis",
      "profiles": ["production"],
      "action": "enable",
      "elevated": true
    }
  ]
}
```

### TOML

```toml
# Base services - always managed (no profiles field)
[[services]]
name = "nginx"
action = "start"
elevated = true

[[services]]
name = "ssh"
action = "enable"
elevated = true

# Development profile services
[[services]]
name = "docker"
profiles = ["dev"]
action = "start"
elevated = true

[[services]]
name = "mysql"
profiles = ["dev"]
action = "enable"
elevated = true

# Production profile services
[[services]]
name = "postgresql"
profiles = ["production"]
action = "start"
elevated = true

[[services]]
name = "redis"
profiles = ["production"]
action = "enable"
elevated = true
```

For more information, see the [Blueprints Overview](../blueprints-general.md) and the [Commands and Flags](../cli/command-and-flags.md) pages.
