# What are Blueprints?

Blueprints are the core of Rinse, Wash, Repeat (RWR) and define how your system should be configured. They are written in YAML, JSON, or TOML format and are processed by RWR to manage various aspects of your system, such as packages, repositories, files, services, and more.

## Blueprint Structure

RWR blueprints are organized into different types, each responsible for managing a specific aspect of your system. The available blueprint types are:

- [Packages](blueprints/packages.md): Manage packages to be installed or removed
- [Repositories](blueprints/repositories.md): Manage repositories for package managers
- [Files](blueprints/files.md): Manage files to be copied, moved, deleted, or created
- [Directories](blueprints/directories.md): Manage directories to be created or deleted
- [Services](blueprints/services.md): Manage system services to be started, stopped, or restarted
- [Configuration](blueprints/configuration.md): Manage configuration settings for your system
- [Git](blueprints/git.md): Manage Git repositories to be cloned or updated
- [Scripts](blueprints/scripts.md): Manage scripts to be executed
- [Users and Groups](blueprints/users-and-groups.md): Manage user accounts and groups

Each blueprint type has its own specific structure and settings, which are described in detail on their respective pages.

## Blueprint Imports

RWR supports importing blueprint definitions from other files, making it easy to share common configurations across multiple systems or projects. Any blueprint entry can use the `import` field to include definitions from an external file.

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

- **Relative Paths**: Import paths are resolved relative to the blueprint directory specified in your init file
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

This approach allows you to:

- Share common configurations across multiple machines
- Maintain DRY (Don't Repeat Yourself) principles
- Organize blueprints logically by separating shared and specific configurations
- Version control shared configurations separately from system-specific ones

For complete examples and best practices, see the [`examples/imports/`](../examples/imports/) directory.

## Blueprint Locations

Blueprints can be stored in a directory specified in the `init.yaml` file under the `blueprints.location` setting. By default, RWR looks for blueprints in a directory named `blueprints` in the same location as the `init.yaml` file.

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

The order in which blueprints are processed is determined by the `blueprints.order` setting in the `init.yaml` file. If not specified, RWR will process blueprints in the following default order:

1. Packages
2. Repositories
3. Files
4. Directories
5. Services
6. Configuration
7. Git
8. Scripts
9. Users and Groups

You can customize the processing order by specifying the desired order in the `blueprints.order` setting. For example:

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

RWR supports the use of variables in blueprints to make them more dynamic and reusable. Variables can be defined in the `init.yaml` file or passed as command-line flags. For more information on using variables in blueprints, refer to the [Variables and Templating](variables.md) page.

## Next Steps

- Explore the specific blueprint type pages to learn more about their structure and settings.
- Learn how to use [Variables and Templating](variables.md) in your blueprints.
- Discover [Best Practices](best-practices.md) for organizing and managing your blueprints.
