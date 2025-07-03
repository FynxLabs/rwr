# Variables and Templating

Rinse, Wash, Repeat (RWR) supports the use of variables and templating in blueprints to make them more dynamic and reusable. This page explains how to use variables and templating in your RWR blueprints.

## Variables

Variables allow you to parameterize your blueprints and make them more flexible. RWR supports two types of variables:

1. User-defined variables
2. Built-in variables

### User-defined Variables

User-defined variables are specified in the `init.yaml` file under the `variables` section. These variables can be referenced in your blueprints using the `{{ .UserDefined.variable_name }}` syntax.

Example `init.yaml` file:

```yaml
variables:
  user_defined:
    app_version: 1.0.0
    server_port: 8080
```

In your blueprint:

```yaml
packages:
  - name: myapp
    version: {{ .UserDefined.app_version }}

services:
  - name: myapp
    port: {{ .UserDefined.server_port }}
```

### Built-in Variables

RWR provides a set of built-in variables that can be used in your blueprints. These variables are automatically populated based on the current system and configuration.

| Variable | Description |
|----------|-------------|
| `{{ rwr.os }}` | The operating system name (e.g., linux, macos, windows) |
| `{{ rwr.arch }}` | The system architecture (e.g., amd64, arm64) |
| `{{ .User.username }}` | The current user's username |
| `{{ .User.firstName }}` | The current user's First Name |
| `{{ .User.lastName }}` | The current user's Last Name |
| `{{ .User.fullName }}` | The current user's home directory |
| `{{ .User.groupName }}` | The current user's Group Name (Linux/macOS Only) |
| `{{ .User.home }}` | The current user's home directory |
| `{{ .User.shell }}` | The current user's shell (e.g.; bash, zsh) |
| `{{ .Flags.debug }}` | Current Debug Flag Setting |
| `{{ .Flags.logLevel }}` | Current Log Level Setting |
| `{{ .Flags.interactive }}` | Current Interactive Mode setting |
| `{{ .Flags.forceBootstrap }}` | Current Force Bootstrap setting |
| `{{ .Flags.ghAPIToken }}` | Current Github API Token |
| `{{ .Flags.sshKey }}` | Current Private SSH Key (base64 encoded) |
| `{{ .Flags.skipVersionCheck }}` | Current Skip Version setting |

## Templating

RWR uses the Go template syntax for templating. You can use templating to conditionally include or exclude sections of your blueprints based on variable values or to generate dynamic content.

### Conditional Sections

You can use the `{{if}}` and `{{end}}` directives to conditionally include or exclude sections of your blueprints.

Example:

```yaml
packages:
  {{if eq rwr.os "linux"}}
  - name: git
    action: install
  {{end}}

  {{if eq rwr.os "macos"}}
  - name: homebrew
    action: install
  {{end}}
```

### Looping

You can use the `{{range}}` and `{{end}}` directives to loop over a list of items.

Example:

```yaml
packages:
  {{range .packages}}
  - name: {{.name}}
    version: {{.UserDefined.version}}
    action: {{.UserDefined.action}}
  {{end}}
```

### Functions

RWR supports a subset of the Go template functions. Here are some commonly used functions:

| Function | Description |
|----------|-------------|
| `{{eq arg1 arg2}}` | Returns true if arg1 and arg2 are equal |
| `{{ne arg1 arg2}}` | Returns true if arg1 and arg2 are not equal |
| `{{lt arg1 arg2}}` | Returns true if arg1 is less than arg2 |
| `{{gt arg1 arg2}}` | Returns true if arg1 is greater than arg2 |
| `{{join list separator}}` | Joins a list of strings with the specified separator |

For a complete list of supported functions, please refer to the [Go template documentation](https://golang.org/pkg/text/template/).

## Best Practices

- Use meaningful variable names that describe the purpose of the variable.
- Keep your templates simple and readable.
- Use variables to avoid hardcoding values in your blueprints.
- Use conditional sections to handle differences between operating systems or configurations.
- Test your templates thoroughly to ensure they work as expected.

By leveraging variables and templating in your RWR blueprints, you can create more flexible and reusable configurations that adapt to different environments and requirements.
