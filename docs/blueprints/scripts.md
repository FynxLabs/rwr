# Scripts Blueprint

The Scripts blueprint allows you to execute scripts as part of the configuration process in Rinse, Wash, Repeat (RWR). This blueprint is useful for running custom scripts, setting up environment-specific configurations, or performing any additional tasks that are not covered by other blueprint types.

## Blueprint Structure

The Scripts blueprint follows the same structure as other blueprints in RWR. It can be defined in YAML, JSON, or TOML format.

### YAML Example

```yaml
scripts:
  # Base script - always runs (no profiles field)
  - name: setup_common
    description: "Common setup script"
    source: scripts/common_setup.sh
    action: run
    exec: bash
    elevated: true
    log: setup

  # Development profile script
  - name: dev_environment
    description: "Setup development environment"
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
    description: "Install work-specific tools"
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
      "description": "Common setup script",
      "source": "scripts/common_setup.sh",
      "action": "run",
      "exec": "bash",
      "elevated": true,
      "log": "setup"
    },
    {
      "name": "dev_environment",
      "description": "Setup development environment",
      "profiles": ["dev"],
      "content": "#!/bin/bash\necho \"Setting up development environment...\"\nexport NODE_ENV=development",
      "action": "run",
      "exec": "bash",
      "args": "--verbose"
    },
    {
      "name": "work_tools",
      "description": "Install work-specific tools",
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
description = "Common setup script"
source = "scripts/common_setup.sh"
action = "run"
exec = "bash"
elevated = true
log = "setup"

# Development profile script
[[scripts]]
name = "dev_environment"
description = "Setup development environment"
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
description = "Install work-specific tools"
profiles = ["work"]
source = "scripts/work_setup.py"
action = "run"
exec = "python"
elevated = false
```

## Blueprint Fields

The Scripts blueprint supports the following fields:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes, if `import` is not provided | The name of the script. |
| `import` | Yes, if `name` is not provided | Path to import script definitions from another file (relative to blueprint directory) |
| `profiles` | No | List of profiles this script belongs to. If empty, script always runs (base item). |
| `action` | Yes | The action to perform with the script. Currently, only `run` is supported. |
| `exec` | No | The script interpreter/executor (e.g., `bash`, `python`, `ruby`, `powershell`, `self`). Auto-detected if not specified. |
| `source` | No | The path to the script file (relative to blueprint directory). |
| `content` | No | The inline content of the script. |
| `args` | No | Additional arguments to pass to the script. |
| `elevated` | No | Whether to run the script with elevated privileges. Default is `false`. |
| `log` | No | Log name for script output. |

> [!NOTE]
> Either the `source`, `content`, or `import` field must be provided. If both `source` and `content` are present, `source` takes precedence.

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

This allows you to reuse common scripts across multiple configurations.

## Script Execution

When the Scripts blueprint is processed, RWR will execute the specified scripts in the order they are defined. The scripts can be provided either as separate files using the `source` field or as inline content using the `content` field.

RWR supports executing scripts written in various languages, such as Bash, Python, Ruby, and more. The appropriate interpreter will be used based on the shebang line (`#!/bin/bash`, `#!/usr/bin/env python`, etc.) or file extension.

If the `elevated` field is set to `true`, the script will be executed with elevated privileges (e.g., using `sudo` on Unix-like systems).

## Best Practices

- Keep your scripts concise and focused on specific tasks.
- Use descriptive names for your scripts to make their purpose clear.
- Provide a shebang line at the beginning of your scripts to specify the interpreter.
- Use the `elevated` field sparingly and only when necessary.
- Consider using variables and templating to make your scripts more dynamic and reusable.
- Test your scripts thoroughly before including them in your RWR configuration.

## Troubleshooting

If you encounter issues with the Scripts blueprint, consider the following:

- Ensure that the script files specified in the `source` field exist and have the correct permissions.
- Verify that the required interpreters or dependencies for your scripts are installed on the target system.
- Check the RWR logs for any error messages or output related to script execution.
- Use the `--debug` flag when running RWR to enable verbose output and gather more information.

If you need further assistance, please refer to the [Troubleshooting](../troubleshooting.md) section or reach out to the RWR community for support.
