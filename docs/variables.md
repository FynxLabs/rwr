# Variables and Templating

You can use variables and templates in RWR blueprints. One blueprint can then serve many machines. This page explains how to use variables and templates in your RWR blueprints.

## Variables

With variables, you can change values without an edit to the blueprint. RWR supports two types of variables:

1. User-defined variables
2. Built-in variables

### User-defined Variables

Give user-defined variables in the init file under `variables.userDefined`.
Read them in a blueprint as `{{ .UserDefined.<key> }}`.

Example `init.yaml` file:

```yaml
variables:
  userDefined:
    app_version: 1.0.0
    editors:
      - vim
      - neovim
```

In your blueprint:

```yaml
packages:
  - name: myapp-{{ .UserDefined.app_version }}
    action: install
    package_manager: pacman
```

`userDefined` is the only part of `variables` you write. RWR fills `User`,
`System`, and `Flags`. You cannot set them from the init file.

#### From the environment

Every environment variable whose name begins with `RWR_` is also placed in
`UserDefined`, under the name with the `RWR_` prefix removed and the rest of the
name unchanged. `RWR_BUILD_ID=42` becomes `{{ .UserDefined.BUILD_ID }}`.

The names are case-sensitive and are not lowercased, so a variable set this way
does not collide with a lower-case key from the init file. A key set in the
environment overwrites a key of exactly the same name from `userDefined`.

> [!NOTE]
> `RWR_` is also the prefix RWR uses for its own [configuration
> options](cli/configuration.md), so `RWR_LOG_LEVEL=debug` both sets the log
> level and appears as `{{ .UserDefined.LOG_LEVEL }}`.

### Built-in Variables

RWR provides a set of built-in variables that you can use in your blueprints. RWR fills these variables from the system and the configuration.

| Variable | Description |
|----------|-------------|
| `{{ .System.os }}` | The operating system: `linux`, `darwin`, or `windows` |
| `{{ .System.osFamily }}` | The distribution family, for example `arch` or `debian` |
| `{{ .System.osVersion }}` | The version of the operating system |
| `{{ .System.osArch }}` | The architecture: `amd64`, `arm64`, or `riscv64` |
| `{{ .User.username }}` | The current user's username |
| `{{ .User.firstName }}` | The current user's First Name |
| `{{ .User.lastName }}` | The current user's Last Name |
| `{{ .User.fullName }}` | The current user's full name |
| `{{ .User.groupName }}` | The current user's Group Name (Linux/macOS Only) |
| `{{ .User.home }}` | The current user's home directory |
| `{{ .User.shell }}` | The current user's shell (for example, bash or zsh) |
| `{{ .Flags.debug }}` | Current Debug Flag Setting |
| `{{ .Flags.logLevel }}` | Current Log Level Setting |
| `{{ .Flags.interactive }}` | Current Interactive Mode setting |
| `{{ .Flags.forceBootstrap }}` | Current Force Bootstrap setting |
| `{{ .Flags.skipVersionCheck }}` | Current Skip Version setting |
| `{{ .Flags.dryRun }}` | Whether dry-run mode is active |
| `{{ .Flags.profiles }}` | The list of active profiles |
| `{{ .Flags.configLocation }}` | The path of the configuration directory |
| `{{ .Flags.runOnceLocation }}` | The path of the run-once directory |

> [!NOTE]
> RWR withholds `{{ .Flags.ghAPIToken }}` and `{{ .Flags.sshKey }}` by default.
> A template writes to a path that the blueprint selects. An exposed credential
> can therefore go to any path. If a blueprint needs one, name it under
> `exposeCredentials`. Read [credentials](credentials.md).

## Templating

RWR uses the Go template syntax. You can use templates to conditionally include or exclude sections of your blueprints, or to generate dynamic content.

### Conditional Sections

You can use the `{{if}}` and `{{end}}` directives to conditionally include or exclude sections of your blueprints.

Example:

```yaml
packages:
  {{if eq .System.os "linux"}}
  - name: git
    action: install
  {{end}}

  {{if eq .System.os "darwin"}}
  - name: homebrew
    action: install
  {{end}}
```

> [!NOTE]
> The value for macOS is `darwin`, not `macos`. RWR uses the name that Go gives
> for the operating system.

### Looping

You can use the `{{range}}` and `{{end}}` directives to read each item in a list.

Only the four groups above are in scope. Put your list in `userDefined` in the
init file, then read it:

```yaml
# init.yaml
variables:
  userDefined:
    editors:
      - vim
      - neovim
```

```yaml
# packages/editors.yaml
packages:
  {{range .UserDefined.editors}}
  - name: {{.}}
    action: install
  {{end}}
```

### Functions

RWR uses the Go template package and adds no functions of its own. These are the
standard functions:

| Function | Description |
|----------|-------------|
| `{{eq arg1 arg2}}` | True when arg1 and arg2 are equal |
| `{{ne arg1 arg2}}` | True when arg1 and arg2 are different |
| `{{lt arg1 arg2}}` | True when arg1 is less than arg2 |
| `{{gt arg1 arg2}}` | True when arg1 is more than arg2 |

CAUTION: RWR registers no additional functions. A template that calls `default`,
`join`, or another function from a different tool stops the run with
`function "default" not defined`.

For the full list of standard functions, read the
[Go template documentation](https://golang.org/pkg/text/template/).

## Best Practices

- Use meaningful variable names that describe the purpose of the variable.
- Keep your templates simple and readable.
- Use variables to avoid hardcoding values in your blueprints.
- Use conditional sections to handle differences between operating systems or configurations.
- Test your templates. Make sure that they give the output that you expect.

With variables and templates, one set of blueprints serves many environments.
