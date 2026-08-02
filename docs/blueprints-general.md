# What are Blueprints?

Blueprints are the core of Rinse, Wash, Repeat (RWR). They define the configuration of your system. You write them in YAML, JSON, or TOML. RWR reads them to manage packages, repositories, files, services, and more.

## Blueprint Structure

RWR blueprints are organized into different types, each responsible for managing a specific aspect of your system. The available blueprint types are:

- [Packages](blueprints/packages.md): Install or remove packages
- [Repositories](blueprints/repositories.md): Manage repositories for package managers
- [Files](blueprints/files.md): Copy, move, delete, or create files
- [Directories](blueprints/directories.md): Create or delete directories
- [Services](blueprints/services.md): Start, stop, or restart system services
- [Configuration](blueprints/configuration.md): Manage the configuration of your system
- [Git](blueprints/git.md): Clone or update Git repositories
- [Scripts](blueprints/scripts.md): Run scripts
- [Users and Groups](blueprints/users-and-groups.md): Manage user accounts and groups

Each blueprint type has its own specific structure and settings, which are described in detail on their respective pages.

## Blueprint Imports

A blueprint can import definitions from another file. You can then share common configurations between systems and projects. Any blueprint entry can use the `import` field to include definitions from an external file.

### Basic Import Syntax

```yaml
packages:
  # Import shared package definitions
  - import: ../../Common/packages/base-packages.yaml
  # Add local packages
  - names:
      - system-specific-package
    action: install
```

### Import Features

- **Relative Paths**: RWR resolves import paths relative to the blueprint directory in your init file
- **Circular Detection**: RWR automatically detects and prevents circular import loops
- **Format Agnostic**: Import files can be in any supported format (YAML, JSON, TOML)
- **Profile Support**: Imported items respect profile filtering just like regular entries
- **All Blueprint Types**: Works with packages, files, services, git repositories, scripts, SSH keys, users, and all other blueprint types

### Import Example

**Shared file** (`Common/packages/base.yaml`):

```yaml
packages:
  - names:
      - git
      - curl
      - vim
    action: install
```

**System-specific file** (`MySystem/packages.yaml`):

```yaml
packages:
  - import: ../../Common/packages/base.yaml
  - names:
      - docker
      - kubectl
    action: install
    profiles: ["dev"]
```

With imports, you can:

- Share common configurations across multiple machines
- Write each definition one time
- Organize blueprints logically, with shared and specific configurations in separate files
- Keep shared configurations in a separate repository

For complete examples and best practices, read the [`examples/imports/`](../examples/imports/) directory.

## Blueprint Locations

Store blueprints in the directory that the `blueprints.location` setting in `init.yaml` gives. By default, RWR looks for blueprints in a directory named `blueprints` in the same location as the `init.yaml` file.

You can organize your blueprints in subdirectories within the main blueprints directory. For example:

```text
blueprints/
  packages/
    common.yaml
    development.yaml
  repositories/
    apt.yaml
    brew.yaml
  files/
    config.yaml
    dotfiles.yaml
  ...
```

## Blueprint Processing Order

The `blueprints.order` setting in the `init.yaml` file gives the order of the
blueprint types. RWR uses this default order when the init file gives no order:

1. Repositories
2. Packages
3. SSH keys
4. Users and groups
5. Files
6. Fonts
7. Services
8. Git
9. Scripts
10. Configuration

Two notes about this list:

- Directories are not a blueprint type. The `directories` key is part of a files
  blueprint, with the `files` and `templates` keys. The files processor reads all
  three keys.
- Package managers are not in this list. RWR installs the package managers from
  the `packageManagers` section of the init file. This happens before the
  blueprint types in the list.

To use a different order, give the order in the `blueprints.order` setting. For
example:

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

## Blueprint Variables

You can use variables in blueprints. One blueprint can then serve many machines. You can define variables in the `init.yaml` file or pass them as command-line flags. For more information, read the [Variables and Templating](variables.md) page.

## Next Steps

- Read the blueprint type pages for their structure and settings.
- Read [Variables and Templating](variables.md) to use variables in your blueprints.
- Read [Best Practices](best-practices.md) for blueprint organization and management.
