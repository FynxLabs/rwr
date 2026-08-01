# Directories Blueprint

The `directories` key manages the directories on your system. You can create,
delete, copy, and move a directory. You can also set the permissions and the
owner of a directory.

> [!IMPORTANT]
> `directories` is not a blueprint type. It is a key in a **files** blueprint,
> with the `files` and `templates` keys. Put it in a file under your `files/`
> directory. The files processor reads all three keys.
>
> There is no `rwr run directories` command. Use `rwr run files`.

## Blueprint Structure

The `directories` key has this structure:

```yaml
directories:
  - name: string
    names: []string
    action: string
    source: string
    target: string
    owner: string
    group: string
    mode: int
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
| `owner` | string | The name of the owner of the directory (for the chown action) |
| `group` | string | The name of the group of the directory (for the chgrp action) |
| `mode` | int | The permissions of the directory, in octal. Write `0755`, not `755`. The value `755` is a decimal number and gives the wrong permissions |
| `elevated` | bool | Whether to perform the action with elevated privileges (default: false) |
| `interactive` | bool | Override global interactive mode for this directory (`true`/`false`). If omitted, uses the global `--interactive` flag. Controls whether diffs are shown before overwriting existing files during copy operations |

## Examples

Here are some examples of using the Directories Blueprint in different formats:

### YAML

```yaml
directories:
  - name: mydir
    action: create
    target: /path/to/mydir
    mode: 0755
    elevated: true

  - names:
      - dir1
      - dir2
    action: copy
    source: /path/to/source
    target: /path/to/destination
    owner: "levi"
    group: "levi"
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
      "owner": "levi",
      "group": "levi"
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
elevated = true

[[directories]]
names = ["dir1", "dir2"]
action = "copy"
source = "/path/to/source"
target = "/path/to/destination"
owner = "levi"
group = "levi"
```

These examples demonstrate how to create a directory with specific permissions, copy multiple directories while setting owner and group, and perform actions with elevated privileges.

For more information on the available actions and their specific requirements, please refer to the [Blueprints Overview](../blueprints-general.md) page.
