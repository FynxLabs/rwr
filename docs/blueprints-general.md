# How blueprints work

A blueprint is a data file that describes part of the workstation you want.
RWR reads the file, selects the matching processor, and brings that resource
toward the declared state.

This package blueprint, for example, asks RWR to install two packages:

```yaml
packages:
  - names: [git, curl]
    action: install
```

Blueprints may use YAML, JSON, TOML, or CUE. The `blueprints.format` field in
the init file selects the format RWR discovers for that tree.

## The init file and the blueprint tree

Every setup starts with an init file:

```yaml
blueprints:
  format: yaml
  location: blueprints
```

`location` points to the directory RWR scans. A small tree can keep every
resource in one file; a larger tree can use folders for operating systems,
machines, or resource types.

```text
my-setup/
├── init.yaml
└── blueprints/
    ├── packages.yaml
    ├── files/
    │   ├── shell.yaml
    │   └── editor.yaml
    └── laptop.yaml
```

Folders and filenames are mainly organizational. RWR recognizes a blueprint by
a processor directory in its path or by its top-level keys. A root-level file
containing both `packages:` and `files:` is routed to both processors.

Payload files belong under a processor's `src/` directory. RWR does not scan
that directory as more blueprint input.

## Blueprint processors

Each processor owns one or more top-level keys:

| Processor | Top-level key | Manages |
|---|---|---|
| `repositories` | `repositories` | Package repositories |
| `packages` | `packages` | Installed and removed packages |
| `ssh_keys` | `ssh_keys` | SSH key generation and GitHub upload |
| `users` | `users`, `groups` | Local users and groups |
| `files` | `files`, `templates`, `directories` | Filesystem content and layout |
| `fonts` | `fonts` | Nerd Fonts |
| `services` | `services` | System services |
| `git` | `git` | Git checkouts |
| `scripts` | `scripts` | Commands outside a native processor |
| `configuration` | `configurations` | dconf, gsettings, macOS defaults, Windows Registry, and Omarchy |
| `credentials` | `credential_setup` | Explicit provider and native credential tasks |

The [blueprint type reference](blueprints/README.md) documents each schema.
Directories and templates are part of the `files` processor, so they are not
separate `rwr run` targets.

## Processing order

Package managers declared in the init file run before blueprint processors.
After that, RWR uses this order unless `blueprints.order` overrides it:

1. Credential setup, when `credentialPolicy.setup` is `ordered`
2. Repositories
3. Packages
4. SSH keys
5. Users and groups
6. Files
7. Fonts
8. Services
9. Git repositories
10. Scripts
11. Configuration values, including post-login Omarchy setup

Credential setup is explicit by default. Run it separately with
`rwr run credentials`, or opt into its ordered position in the
[credential policy](credentials.md#acquisition-policy).

To set a custom order:

```yaml
blueprints:
  format: yaml
  location: blueprints
  order:
    - repositories
    - packages
    - files
    - services
```

Use a flat list when sequence matters. `packageManagers` does not belong in
this list because RWR handles it before processor dispatch.

## Shared entries and profiles

An entry can import entries from another file:

```yaml
packages:
  - import: ../shared/base-packages.yaml
  - name: podman
    action: install
    profiles: [development]
```

Import paths are relative to the file that declares them. Imports may be nested;
RWR reports missing files and circular chains during validation. See
[common fields](blueprints/common-fields.md#import) for the processors that
support imports.

Profiles let one tree serve several machines or roles. Entries without a
`profiles` field form the shared base. Once you pass `--profile`, RWR adds only
entries matching an active profile:

```bash
rwr all --profile development
```

With no `--profile` flag, RWR applies all entries, including profiled ones. Read
[Profiles](profiles.md) before relying on profiles to narrow a run.

## Variables and CUE

Go template expressions can use values from the current user, system, CLI
flags, and the init file:

```yaml
files:
  - name: config.toml
    action: create
    target: "{{ .User.home }}/.config/example"
    content: "channel = {{ .UserDefined.release_channel }}"
```

The [variables and templates guide](variables.md) lists the available values.

CUE blueprints add constraints and composition. RWR evaluates CUE itself, so a
`cue` binary is not required on the target machine. CUE imports are limited to
its built-in standard library; filesystem and network imports are refused.

## Check a tree before applying it

Use the same sequence for new resources and later changes:

```bash
rwr validate PATH_TO_TREE
rwr all --init-file PATH_TO_INIT --dry-run
rwr all --init-file PATH_TO_INIT
```

Validation catches unsupported fields, invalid schema versions, broken imports,
and template errors before a run changes the machine.

Continue with the [init file reference](init-file.md), then choose a page from
the [blueprint type reference](blueprints/README.md).
