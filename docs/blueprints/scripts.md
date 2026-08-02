# Scripts Blueprint

With the Scripts blueprint, you can run scripts as part of the configuration process in Rinse, Wash, Repeat (RWR). This blueprint runs custom scripts and does tasks that other blueprint types do not cover.

## Blueprint Structure

The Scripts blueprint follows the same structure as other blueprints in RWR. It can be defined in YAML, JSON, or TOML format.

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

### YAML Example

```yaml
scripts:
  # Base script - always runs (no profiles field)
  - name: setup_common
    source: scripts/common_setup.sh
    action: run
    exec: bash
    elevated: true
    log: setup

  # Development profile script
  - name: dev_environment
    profiles:
      - dev
    content: |
      #!/bin/bash
      echo "Setting up development environment..."
      export NODE_ENV=development
    action: run
    exec: bash
    args: "--verbose"

  # Work profile script with custom executor
  - name: work_tools
    profiles:
      - work
    source: scripts/work_setup.py
    action: run
    exec: python
    elevated: false
```

### JSON Example

```json
{
  "scripts": [
    {
      "name": "setup_common",
      "source": "scripts/common_setup.sh",
      "action": "run",
      "exec": "bash",
      "elevated": true,
      "log": "setup"
    },
    {
      "name": "dev_environment",
      "profiles": ["dev"],
      "content": "#!/bin/bash\necho \"Setting up development environment...\"\nexport NODE_ENV=development",
      "action": "run",
      "exec": "bash",
      "args": "--verbose"
    },
    {
      "name": "work_tools",
      "profiles": ["work"],
      "source": "scripts/work_setup.py",
      "action": "run",
      "exec": "python",
      "elevated": false
    }
  ]
}
```

### TOML Example

```toml
# Base script - always runs (no profiles field)
[[scripts]]
name = "setup_common"
source = "scripts/common_setup.sh"
action = "run"
exec = "bash"
elevated = true
log = "setup"

# Development profile script
[[scripts]]
name = "dev_environment"
profiles = ["dev"]
content = """
#!/bin/bash
echo "Setting up development environment..."
export NODE_ENV=development
"""
action = "run"
exec = "bash"
args = "--verbose"

# Work profile script with custom executor
[[scripts]]
name = "work_tools"
profiles = ["work"]
source = "scripts/work_setup.py"
action = "run"
exec = "python"
elevated = false
```

## Blueprint Settings

The Scripts blueprint supports the following settings:

| Setting | Required | Description |
|-------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the script. |
| `import` | Yes, if `name` is not provided | Path to import script definitions from another file (relative to blueprint directory) |
| `profiles` | No | List of profiles this script belongs to. If empty, script always runs (base item). |
| `action` | Yes | The action to perform with the script. Currently, only `run` is supported. |
| `exec` | No | The program that runs the script: `bash`, `python`, `ruby`, `perl`, `lua`, `powershell`, or `self`. The default is `bash` on Linux and macOS, and `powershell` on Windows. `self` runs the script file directly |
| `source` | No | The path to the script file (relative to blueprint directory). |
| `content` | No | The inline content of the script. |
| `args` | No | Additional arguments to pass to the script. |
| `elevated` | No | Whether to run the script with elevated privileges (`sudo`). Default is `false`. |
| `asUser` | No | Run the script as another account: `sudo -u <user>`. Ignored when `elevated: true`, since sudo cannot do both at once. RWR warns and runs elevated. |
| `log` | No | Log name for script output. |
| `interactive` | No | Override global interactive mode for this script (`true`/`false`). If omitted, uses the global `--interactive` flag. |

> [!NOTE]
> Either the `source`, `content`, or `import` field must be provided. If both `source` and `content` are present, `source` takes precedence.

### How RWR runs the arguments

RWR divides the `args` string at each space. It then sends each part to the
script as one argument. The value `--verbose --out /tmp` becomes three
arguments.

RWR does not use a shell to run the script. The shell characters keep their
literal value. These characters have no special function:

| Character | Result |
|---|---|
| `$HOME`, `$(command)` | RWR sends the text without a change |
| `~` | RWR sends the character without a change |
| `*`, `?` | RWR does not expand the pattern to file names |
| `&&`, `\|`, `;` | RWR sends the character as part of the argument |

To use a shell function, run a shell as the script. Give the command in the
script file, or run the shell directly:

```yaml
scripts:
  - name: build.sh
    action: run
    exec: bash
    source: ./scripts/
```

The script file can then use `$HOME`, pipes, and the other shell functions.

## Blueprint Imports

Import script definitions from other files:

```yaml
scripts:
  # Import common setup scripts
  - import: ../../Common/scripts/base-setup.yaml

  # Add environment-specific scripts
  - name: custom_setup
    content: |
      #!/bin/bash
      echo "Running custom setup..."
    action: run
    exec: bash
    profiles:
      - dev
```

You can then reuse common scripts across multiple configurations.

## Script Execution

RWR runs the scripts in the order that the blueprint defines them. You can give a script as a separate file with the `source` field or as inline content with the `content` field.

RWR runs scripts in Bash, Python, Ruby, Perl, Lua, and PowerShell. The `exec`
field gives the program that runs the script.

CAUTION: RWR does not read the shebang line and does not read the file
extension. Give the `exec` field. Without it, RWR uses `bash` on Linux and
macOS, and `powershell` on Windows.

With `elevated: true`, RWR runs the script as `sudo -- <program> <script>`. With
the `asUser` field, RWR runs it as `sudo -u <user> -- <program> <script>`:

```yaml
scripts:
  - name: build_aur_package
    action: run
    exec: bash
    source: ./scripts/
    asUser: builder
```

If you declare both `elevated: true` and `asUser`, RWR runs the script elevated
and ignores `asUser`. A warning names the dropped account.

## Best Practices

- Keep your scripts concise and focused on specific tasks.
- Use descriptive names for your scripts to make their purpose clear.
- Give the `exec` field for each script. RWR does not read the shebang line.
- Use the `elevated` field only when the script requires elevation.
- Use variables and templating to make your scripts reusable.
- Test your scripts before you include them in your RWR configuration.

## Troubleshooting

If the Scripts blueprint fails, make these checks:

- Make sure that the script files in the `source` field exist and have the correct permissions.
- Make sure that the interpreters or dependencies for your scripts are installed on the target system.
- Examine the RWR logs for error messages or output from the script run.
- Use the `--debug` flag when you run RWR to get verbose output and more information.

If you need further assistance, open an issue at [github.com/fynxlabs/rwr/issues](https://github.com/fynxlabs/rwr/issues).
