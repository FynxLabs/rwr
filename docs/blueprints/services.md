# Services Blueprint

The Services Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage system services, including starting, stopping, enabling, and disabling services across different operating systems.

## Blueprint Structure

The Services Blueprint is defined in a YAML, JSON, or TOML file and consists of an array of service objects. Each service object represents a system service and its associated properties.

```yaml
services:
  - name: nginx
    action: start
    elevated: true
  - name: mysql
    action: stop
    elevated: true
```

## Service Object Properties

Each service object in the Services Blueprint can have the following properties:

| Property | Required | Description |
|----------|----------|-------------|
| `name` | Yes | The name of the service |
| `profiles` | No | List of profiles this service belongs to. If empty, service is always managed (base item) |
| `action` | Yes | The action to perform on the service (start, stop, enable, disable, restart, reload, status, create, delete) |
| `elevated` | No | Whether the service requires elevated privileges (default: false) |
| `target` | No | The target file for the service (used with create and delete actions) |
| `content` | No | The content of the service file (used with the create action) |
| `source` | No | The source file for the service (used with the create action) |
| `file` | No | The file associated with the service (used with the delete action) |

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

## Platform-Specific Considerations

The Services Blueprint handles service management differently depending on the operating system:

### Linux (systemd)

On Linux systems with systemd, the Services Blueprint uses the `systemctl` command to manage services. The `create` and `delete` actions manage service unit files in the appropriate systemd directories.

### macOS (launchd)

On macOS, the Services Blueprint uses the `launchctl` command to manage services. The `create` and `delete` actions manage service plist files in the `/Library/LaunchDaemons` directory.

### Windows

On Windows, the Services Blueprint uses the `sc` command to manage services. The `create` and `delete` actions manage service configuration and binaries.

## Examples

Here are a few examples of using the Services Blueprint in different formats:

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

For more information on using the Services Blueprint in your RWR configuration, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Commands and Flags](../cli/command-and-flags.md) pages.
