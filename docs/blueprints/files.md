# The Files Blueprint

The Files Blueprint in Rinse, Wash, Repeat (RWR) allows you to manage files on your system. You can copy, move, delete, create, and modify files using this blueprint type. Additionally, the Files Blueprint includes the functionality of the former Templates Blueprint, allowing you to process and render template files.

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
| `names` | Yes, if `name` or `import` is not provided | A list of file names to manage (allows batch operations). |
| `import` | Yes, if `name` or `names` is not provided | Path to import file/template definitions from another file (relative to blueprint directory) |
| `profiles` | No | List of profiles this file/template belongs to. If empty, file is always processed (base item). |
| `action` | Yes | The action to perform on the file or template. Valid values are `copy`, `move`, `delete`, `create`, `chmod`, `chown`, `chgrp`, and `symlink`. |
| `source` | No | The source path or URL of the file or template. Required for `copy` and `move` actions. Can be a local path or a URL. |
| `target` | Yes | The target path of the file or template. If it doesn't end with a '/', it's considered a rename operation. |
| `content` | No | The content of the file. Used for the `create` action. |
| `owner` | No | The owner of the file or template. Used for the `chown` action. |
| `group` | No | The group of the file or template. Used for the `chgrp` action. |
| `mode` | No | The file mode in octal notation. Used for the `chmod` action. |
| `elevated` | No | Whether to perform the action with elevated privileges. Defaults to `false`. |
| `variables` | No | A map of variables and their values to be used for template rendering. Only applicable to the `templates` section. |

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

The `files` section of the blueprint is used to manage regular files. The specified actions (`copy`, `move`, `delete`, `create`, `chmod`, `chown`, `chgrp`) are performed on the files without any additional processing.

### URL Sources

RWR now supports URL sources for files. If the `source` field is a URL, RWR will download the file from the specified URL before performing the requested action.

### Intelligent Renaming

When the `target` doesn't end with a '/', RWR considers it a rename operation. Additionally, RWR implements intelligent renaming by attempting to find a file with a similar name (case-insensitive) if the exact name isn't found in the source directory.

## Template Processing

The `templates` section of the blueprint is used to process and render template files. The specified actions are performed on the template files, and the `source` file is treated as a template and rendered with the provided variables.

Template files use the Go template syntax and can include variables, conditionals, and loops. The `variables` setting allows you to define a map of variables and their corresponding values, which can be used within the template files.

> [!NOTE]
> The `variables` setting is only applicable to the `templates` section and will not be used for regular files defined in the `files` section.

For more information on using variables and templating in RWR, please refer to the [Variables and Templating](../variables.md) documentation.

## Examples

Here are some examples of using the Files Blueprint for both regular files and templates in YAML, JSON, and TOML formats:

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
    mode: 0755
    elevated: true

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
    mode: 0755
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
    owner: root
    group: root
    mode: 0644
    elevated: true
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
      "mode": 493,
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
      "mode": 493,
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
      "owner": "root",
      "group": "root",
      "mode": 420,
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
mode = 0o755
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
mode = 0o755
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
owner = "root"
group = "root"
mode = 0o644
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
    mode: 0644
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
    mode: 0644
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
    mode: 0644
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
      "mode": 420,
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
mode = 0o644
[templates.variables]
server_name = "example.com"
port = 80
```

## Notes

- The `chmod`, `chown`, and `chgrp` actions require elevated privileges to modify file permissions and ownership.
- When using the `create` action, the target directory must exist. RWR will not automatically create missing directories.
- File modes for the `chmod` action can be specified in octal notation (e.g., `0644`) or as a decimal value (e.g., `755`).
- When using URL sources, RWR will download the file to a temporary location before performing the requested action.
- Intelligent renaming allows for more flexible file management, especially when dealing with files that might have slight variations in naming across different systems.

For more information on using the Files Blueprint in your RWR configuration, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) guide.
