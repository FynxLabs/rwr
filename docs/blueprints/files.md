# The Files Blueprint

With the Files Blueprint in Rinse, Wash, Repeat (RWR), you manage files on your system. You can copy, move, delete, create, and modify files with this blueprint type. The Files Blueprint also renders template files, the function of the former Templates Blueprint.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Blueprint Structure

The Files Blueprint has the following structure:

```yaml
files:
  - name: <file_name>
    action: <action>
    source: <source_path_or_url>
    target: <target_path>
    content: <file_content>
    owner: <owner>
    group: <group>
    mode: <mode>
    elevated: <elevated>

templates:
  - name: <template_name>
    action: <action>
    source: <source_path_or_url>
    target: <target_path>
    owner: <owner>
    group: <group>
    mode: <mode>
    elevated: <elevated>
    variables:
      <variable_name>: <variable_value>
```

## Blueprint Settings

The following settings are available for the Files Blueprint:

| Setting | Required | Description |
|---------|----------|-------------|
| `name` | Yes, if `names` or `import` is not provided | The name of the file or template. |
| `names` | Yes, if `name` or `import` is not provided | A list of file names to manage (for batch operations). |
| `import` | Yes, if `name` or `names` is not provided | Path to import file/template definitions from another file (relative to blueprint directory) |
| `profiles` | No | List of profiles this file/template belongs to. If empty, file is always processed (base item). |
| `action` | Yes | The action to perform on the file or template. Valid values are `copy`, `move`, `delete`, `create`, `chmod`, `chown`, `chgrp`, and `symlink`. |
| `source` | No | The source path or URL of the file or template. Required for `copy`, `move` and `symlink`. Can be a local path or a URL. |
| `target` | Yes | Where the file goes. See [Target paths](#target-paths). |
| `content` | No | The content of the file. Setting it forces the `create` action. |
| `owner` | No | The owner of the file. Applied by `chown`, and after a `create`. |
| `group` | No | The group of the file. Applied by `chown`/`chgrp`, and after a `create`. |
| `mode` | No | The permission mode. Applied by `create` and `chmod` only. See [File modes](#file-modes). |
| `elevated` | No | Only the `copy` action reads this. With `true`, the copy is staged through a temporary file and installed with elevation. Every other file action is performed by rwr's own process and ignores the field. Defaults to `false`. |
| `variables` | No | A map of variables and their values for template rendering. Only applicable to the `templates` section. |
| `interactive` | No | Override global interactive mode for this file (`true`/`false`). If omitted, uses the global `--interactive` flag. Controls whether RWR shows diffs before it overwrites existing files. |

## Blueprint Imports

Import file and template definitions from other blueprint files:

```yaml
files:
  # Import common dotfiles
  - import: ../../Common/files/dotfiles.yaml

  # Add system-specific files
  - name: local-config.ini
    action: create
    target: /etc/myapp/
    content: |
      [settings]
      local=true

directories:
  # Import shared directory structure
  - import: ../shared/directories.yaml

templates:
  # Import common templates
  - import: ../../Common/templates/configs.yaml
```

Import features work for files, directories, and templates within the Files Blueprint.

## File Processing

The `files` section manages regular files. The processor dispatches on `action`:

| Action | What it does |
|--------|--------------|
| `copy` | Copies `<blueprint_dir>/<source>/<name>` to the target |
| `move` | Renames the source to the target. A move already carried out is not an error |
| `delete` | Removes the target. A target already absent is not an error |
| `create` | Writes `content` to the target and creates missing parent directories |
| `chmod` | Applies `mode` to the target. `mode` is required |
| `chown` | Applies `owner` and/or `group` to the target |
| `chgrp` | Applies `group` to the target |
| `symlink` | Makes the target a symlink to the source. RWR leaves an existing correct link alone and replaces an existing wrong link. An existing regular file or directory is an error |

There is no `append` action and no `template` action. Templates are a separate
section, described below.

Setting `content` forces the action to `create`. If the entry declared something
else, RWR logs a warning and creates the file anyway.

### URL Sources

If `source` is a URL, RWR downloads it to a temporary directory first and then
performs the requested action on the downloaded file.

### Target paths

RWR resolves `target` to the single path the action operates on:

- A target ending in `/` names the **directory** the file goes into: RWR
  appends the entry's `name` to it. `target: /etc/myapp/` with `name: app.conf`
  writes `/etc/myapp/app.conf`.
- Any other target is the **file's own path**, which is how you rename on the
  way: `target: /etc/myapp/production.conf`.
- **Exception:** if the target already exists and is a directory, RWR appends
  the name to it even without the trailing separator. The alternative is
  truncation of a directory, and no blueprint intends that.

RWR expands `~`. Expansion drops a trailing separator internally. Thus RWR
reads the separator from the value you wrote, and `target: "~/.config/"` still
behaves as a directory.

## File modes

`mode` is a permission mode, at most four octal digits (`0o7777`). Only the
`create` and `chmod` actions apply it. A copy takes the permissions of the
source. A symlink has none. The delete, chown and chgrp actions do not touch
the mode. `rwr validate` warns when a mode is set on an action that ignores it.

### How to write one

**Write it as a quoted octal string.** `mode: "0644"` means the same thing in
YAML, JSON and TOML, and needs no thought about how each format reads numbers.
RWR accepts the `0` and `0o` prefixes, and no prefix at all: `"0644"`,
`"0o644"` and `"644"` are the same mode.

RWR reads a number as **the mode's own value**, which is what every parser
already produces for an octal literal. YAML `0644`, TOML `0o644` and JSON `420`
are one and the same mode, `0o644`.

RWR **refuses** a bare integer that reads like octal digits typed without
quotes:

```yaml
mode: 644     # ERROR
```

```text
ambiguous file mode 644: as a number it is 0o1204, but it reads like the octal
mode 0644 typed without quotes; write mode: "0644" for 0o644, or mode: "1204"
for 0o1204
```

This is the change to watch for. `mode: 644` used to decode to 0o1204 — a setuid
mode nobody asked for — and apply it without complaint. It is now an error, and
the message tells you what to write instead. RWR refuses anything above
`0o7777`, or a negative number, the same way.

### TOML and setuid/setgid/sticky modes

TOML's decoder hands rwr the decoded number with the literal already lost. Thus
`0o4755` and `2541` arrive identically, and the ambiguity check cannot tell
them apart. In TOML, **a mode with a setuid, setgid or sticky bit must be a quoted
string**:

```toml
mode = "04755"   # correct
mode = 0o4755    # ERROR: ambiguous file mode 2541
```

Plain permission modes at or below `0o777` are unaffected: `mode = 0o644` is
fine in TOML.

### Defaults

When no `mode` is declared:

| Entry | Mode |
|-------|------|
| A rendered template | `0600` |
| A plain file (`create`) | `0644` |
| A directory | `0755` |

Rendered templates are private because templates can render credentials (a
`.netrc`, a `gh` config). Credentials must not exist world-readable, even for
an instant. For the same reason, when a created file's mode gives nothing to
group or other, RWR makes new parent directories `0700`, not `0755`.

A `chmod` action with no `mode` is an error at validation time and at run time.
RWR never treats it as `0000`.

## Template Processing

The `templates` section of the blueprint renders template files. RWR reads the
template from `<blueprint_dir>/<source>/<name>`, renders it with the merged
variable set, and writes it to `target`.

A template entry needs `name`, `source` and `target`. RWR skips an entry that
misses any of them and logs a warning. RWR writes the rendered result as
content. Thus a template is always **created** at its target, whatever `action`
says. An entry that declares `action: copy` renders, creates, and logs a
warning.

Template files use the Go template syntax and can include variables, conditionals, and loops. With the `variables` setting, you define a map of variables and their values for use in the template files.

> [!NOTE]
> The `variables` setting applies to the `templates` section only. RWR does not use it for regular files in the `files` section.

For more information, see the [Variables and Templating](../variables.md) page.

## Examples

Examples for regular files and templates in YAML, JSON, and TOML:

### Regular Files

#### YAML

```yaml
files:
  # Base files - always processed (no profiles field)
  - name: global-config.ini
    action: copy
    source: ./config/
    target: /etc/myapp/
    elevated: true

  - names:
      - common.sh
      - utils.sh
    action: copy
    source: ./scripts/
    target: /usr/local/bin/
    elevated: true

  # A copy takes the source's permissions; set them with a chmod entry
  - name: common.sh
    action: chmod
    target: /usr/local/bin/common.sh
    mode: "0755"

  # Development profile files
  - name: dev-config.json
    profiles:
      - dev
    action: create
    target: /home/user/.config/myapp/
    content: |
      {
        "debug": true,
        "log_level": "debug"
      }

  - names:
      - dev-script.sh
      - debug-tools.sh
    profiles:
      - dev
    action: copy
    source: ./dev-scripts/
    target: /usr/local/bin/
    elevated: true

  # Production profile files
  - name: production.conf
    profiles:
      - production
    action: copy
    source: https://config.example.com/prod.conf
    target: /etc/myapp/production.conf
    elevated: true

  - name: ssl-cert.pem
    profiles:
      - production
    action: copy
    source: ./certs/
    target: /etc/ssl/certs/
    elevated: true

  # A copy does not apply owner or group either; use a chown entry
  - name: ssl-cert.pem
    profiles:
      - production
    action: chown
    target: /etc/ssl/certs/ssl-cert.pem
    owner: root
    group: root
```

#### JSON

```json
{
  "files": [
    {
      "name": "global-config.ini",
      "action": "copy",
      "source": "./config/",
      "target": "/etc/myapp/",
      "elevated": true
    },
    {
      "names": ["common.sh", "utils.sh"],
      "action": "copy",
      "source": "./scripts/",
      "target": "/usr/local/bin/",
      "elevated": true
    },
    {
      "name": "dev-config.json",
      "profiles": ["dev"],
      "action": "create",
      "target": "/home/user/.config/myapp/",
      "content": "{\n  \"debug\": true,\n  \"log_level\": \"debug\"\n}"
    },
    {
      "names": ["dev-script.sh", "debug-tools.sh"],
      "profiles": ["dev"],
      "action": "copy",
      "source": "./dev-scripts/",
      "target": "/usr/local/bin/",
      "elevated": true
    },
    {
      "name": "production.conf",
      "profiles": ["production"],
      "action": "copy",
      "source": "https://config.example.com/prod.conf",
      "target": "/etc/myapp/production.conf",
      "elevated": true
    },
    {
      "name": "ssl-cert.pem",
      "profiles": ["production"],
      "action": "copy",
      "source": "./certs/",
      "target": "/etc/ssl/certs/",
      "elevated": true
    }
  ]
}
```

#### TOML

```toml
# Base files - always processed (no profiles field)
[[files]]
name = "global-config.ini"
action = "copy"
source = "./config/"
target = "/etc/myapp/"
elevated = true

[[files]]
names = ["common.sh", "utils.sh"]
action = "copy"
source = "./scripts/"
target = "/usr/local/bin/"
elevated = true

# Development profile files
[[files]]
name = "dev-config.json"
profiles = ["dev"]
action = "create"
target = "/home/user/.config/myapp/"
content = """
{
  "debug": true,
  "log_level": "debug"
}
"""

[[files]]
names = ["dev-script.sh", "debug-tools.sh"]
profiles = ["dev"]
action = "copy"
source = "./dev-scripts/"
target = "/usr/local/bin/"
elevated = true

# Production profile files
[[files]]
name = "production.conf"
profiles = ["production"]
action = "copy"
source = "https://config.example.com/prod.conf"
target = "/etc/myapp/production.conf"
elevated = true

[[files]]
name = "ssl-cert.pem"
profiles = ["production"]
action = "copy"
source = "./certs/"
target = "/etc/ssl/certs/"
elevated = true
```

### Templates

#### Templates YAML

```yaml
templates:
  # Base template - always processed (no profiles field)
  - name: app.conf
    action: create
    source: ./templates/
    target: /etc/myapp/app.conf
    owner: root
    group: root
    mode: "0644"
    elevated: true
    variables:
      app_name: MyApplication
      log_level: info

  # Development profile template
  - name: nginx-dev.conf
    profiles:
      - dev
    action: create
    source: ./templates/
    target: /etc/nginx/sites-available/dev.conf
    owner: root
    group: root
    mode: "0644"
    elevated: true
    variables:
      server_name: dev.example.com
      port: 3000
      debug: true

  # Production profile template
  - name: nginx-prod.conf
    profiles:
      - production
    action: create
    source: ./templates/
    target: /etc/nginx/sites-available/prod.conf
    owner: root
    group: root
    mode: "0644"
    elevated: true
    variables:
      server_name: example.com
      port: 80
      ssl_enabled: true
```

#### Templates JSON

```json
{
  "templates": [
    {
      "name": "nginx.conf",
      "action": "create",
      "source": "/path/to/templates/nginx.conf.tmpl",
      "target": "/etc/nginx/nginx.conf",
      "owner": "root",
      "group": "root",
      "mode": "0644",
      "variables": {
        "server_name": "example.com",
        "port": 80
      }
    }
  ]
}
```

#### Templates TOML

```toml
[[templates]]
name = "nginx.conf"
action = "create"
source = "/path/to/templates/nginx.conf.tmpl"
target = "/etc/nginx/nginx.conf"
owner = "root"
group = "root"
mode = "0644"
[templates.variables]
server_name = "example.com"
port = 80
```

## Notes

- rwr's own process carries out the `chmod`, `chown`, and `chgrp` actions, not a helper. Changing a file you do not own therefore needs rwr itself to run with the privilege to do it.
- The `create` action creates missing parent directories.
- Write `mode` as a quoted octal string, `mode: "0644"`. A bare `mode: 644` is an error. See [File modes](#file-modes).
- When the source is a URL, RWR downloads the file to a temporary location before it performs the action.
- Every action is safe to re-run: a delete of an absent file, a move already made, and a symlink already pointing at the source all succeed without changing anything.

For more information, see the [Blueprints Overview](../blueprints-general.md) and [Best Practices](../best-practices.md).
