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

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

The `directories` key has this structure:

```yaml
directories:
  - name: string
    profiles: []string
    action: string
    source: string
    target: string
    owner: string
    group: string
    mode: string
    elevated: bool
```

## Blueprint Settings

The following settings are available for the Directories Blueprint:

| Setting | Type | Description |
|---------|------|-------------|
| `name` | string | The name of the directory. It is appended to `target` for every action |
| `names` | []string | Accepted by the schema but **not used**: the processor reads `name` only. Write one entry per directory |
| `profiles` | []string | Profiles this directory belongs to. Empty means it is always processed |
| `import` | string | Path to import directory definitions from another file, relative to the blueprint directory |
| `action` | string | The action to perform (`create`, `delete`, `copy`, `move`, `chmod`, `chown`, `chgrp`, `symlink`) |
| `source` | string | The source directory, relative to the blueprint directory (for `copy`, `move`, and `symlink`) |
| `target` | string | The parent directory. The entry's `name` is joined onto it, so `target: ~/` with `name: .config` manages `~/.config`. The one exception is `symlink`, where `target` is the link's own path |
| `owner` | string | The owner of the directory (applied by `chown`, and after `create` and `copy`) |
| `group` | string | The group of the directory (applied by `chown`/`chgrp`, and after `create` and `copy`) |
| `mode` | string | The permissions of the directory. Write a quoted octal string: `mode: "0755"`. A bare `mode: 755` is an **error** — see [File modes](files.md#file-modes). Defaults to `0755` when omitted; required for `chmod` |
| `elevated` | bool | Read by `copy` only; other directory actions are performed by rwr's own process (default: false) |
| `interactive` | bool | Override global interactive mode for this directory (`true`/`false`). If omitted, uses the global `--interactive` flag. Controls whether diffs are shown before overwriting existing files during copy operations |

## Examples

Here are some examples of using the Directories Blueprint in different formats:

### YAML

```yaml
directories:
  - name: mydir
    action: create
    target: /path/to/
    mode: "0755"
    elevated: true

  - name: dir1
    action: copy
    source: ./src
    target: /path/to/destination/
    owner: "levi"
    group: "levi"

  - name: .config
    profiles:
      - dev
    action: create
    target: "{{ .User.home }}/"
    mode: "0700"
```

### JSON

```json
{
  "directories": [
    {
      "name": "mydir",
      "action": "create",
      "target": "/path/to/",
      "mode": "0755",
      "elevated": true
    },
    {
      "name": "dir1",
      "action": "copy",
      "source": "./src",
      "target": "/path/to/destination/",
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
target = "/path/to/"
mode = "0755"
elevated = true

[[directories]]
name = "dir1"
action = "copy"
source = "./src"
target = "/path/to/destination/"
owner = "levi"
group = "levi"
```

These examples demonstrate how to create a directory with specific permissions, copy a directory while setting owner and group, and perform actions with elevated privileges.

For more information on the available actions and their specific requirements, please refer to the [Blueprints Overview](../blueprints-general.md) page.
