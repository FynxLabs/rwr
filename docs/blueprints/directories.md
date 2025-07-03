# Directories Blueprint

The Directories Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage directories on your system. You can create, delete, copy, move, and set permissions and ownership for directories.

## Blueprint Structure

The Directories Blueprint has the following structure:

```yaml
directories:
  - name: string
    names: []string
    action: string
    source: string
    target: string
    owner: int
    group: int
    mode: int
    create: bool
    elevated: bool
```

## Blueprint Settings

The following settings are available for the Directories Blueprint:

| Setting | Type | Description |
|---------|------|-------------|
| `name` | string | The name of the directory |
| `names` | []string | An array of directory names to perform the action on |
| `action` | string | The action to perform on the directory (create, delete, copy, move, chmod, chown, chgrp, symlink) |
| `source` | string | The source directory path (for copy, move, and symlink actions) |
| `target` | string | The target directory path |
| `owner` | int | The user ID of the directory owner (for chown action) |
| `group` | int | The group ID of the directory group (for chgrp action) |
| `mode` | int | The permissions of the directory in octal notation (for chmod action) |
| `create` | bool | Whether to create the parent directories if they don't exist (default: false) |
| `elevated` | bool | Whether to perform the action with elevated privileges (default: false) |

## Examples

Here are some examples of using the Directories Blueprint in different formats:

### YAML

```yaml
directories:
  - name: mydir
    action: create
    target: /path/to/mydir
    mode: 0755
    create: true
    elevated: true

  - names:
      - dir1
      - dir2
    action: copy
    source: /path/to/source
    target: /path/to/destination
    owner: 1000
    group: 1000
```

### JSON

```json
{
  "directories": [
    {
      "name": "mydir",
      "action": "create",
      "target": "/path/to/mydir",
      "mode": 493,
      "create": true,
      "elevated": true
    },
    {
      "names": [
        "dir1",
        "dir2"
      ],
      "action": "copy",
      "source": "/path/to/source",
      "target": "/path/to/destination",
      "owner": 1000,
      "group": 1000
    }
  ]
}
```

### TOML

```toml
[[directories]]
name = "mydir"
action = "create"
target = "/path/to/mydir"
mode = 493
create = true
elevated = true

[[directories]]
names = ["dir1", "dir2"]
action = "copy"
source = "/path/to/source"
target = "/path/to/destination"
owner = 1000
group = 1000
```

These examples demonstrate how to create a directory with specific permissions, copy multiple directories while setting owner and group, and perform actions with elevated privileges.

For more information on the available actions and their specific requirements, please refer to the [Blueprints Overview](../blueprints-general.md) page.
