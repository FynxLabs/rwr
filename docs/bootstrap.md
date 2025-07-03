# Bootstrap Process

The Bootstrap Process in Rinse, Wash, Repeat (RWR) is responsible for setting up the initial system configuration. It ensures that the necessary prerequisites and dependencies are in place before executing the main blueprints. This page explains how the Bootstrap Process works and how to define the bootstrap file.

## Overview

The Bootstrap Process is executed before any other blueprints are processed. It typically includes tasks such as:

- Installing essential packages and tools
- Setting up package managers
- Creating required directories and files
- Configuring system settings and permissions
- Setting up SSH keys

The Bootstrap Process is defined in a separate blueprint file named `bootstrap.yaml` (or `bootstrap.json` or `bootstrap.toml`, depending on the chosen format).

## Bootstrap File Structure

The structure of the bootstrap file is similar to other blueprint files in RWR. It can include the following sections:

- `packages`: Defines the packages to be installed during the bootstrap process.
- `files`: Specifies the files to be created or modified during the bootstrap process.
- `directories`: Defines the directories to be created during the bootstrap process.
- `git`: Specifies the Git repositories to be cloned during the bootstrap process.
- `services`: Defines the services to be managed (started, stopped, enabled, disabled) during the bootstrap process.
- `users`: Specifies the user accounts to be created during the bootstrap process.
- `groups`: Defines the groups to be created during the bootstrap process.
- `ssh_keys`: Specifies the SSH keys to be generated during the bootstrap process.

Here's an example of a `bootstrap.yaml` file:

```yaml
packages:
  - name: git
    action: install
  - name: curl
    action: install
  - names:
      - vim
      - tmux
    action: install
    package_manager: apt
    args: ["--no-install-recommends"]

files:
  - name: config.ini
    action: create
    content: |
      [settings]
      debug = true
  - name: remote-config.txt
    action: copy
    source: https://example.com/remote-config.txt
    target: /etc/app/config.txt

directories:
  - name: data
    action: create
    mode: 0755

git:
  - name: my-repo
    action: clone
    url: https://github.com/example/my-repo.git
    path: /opt/my-repo

services:
  - name: nginx
    action: enable

users:
  - name: johndoe
    action: create
    password: "$6$mysecretpassword"
    groups:
      - sudo
      - docker

groups:
  - name: developers
    action: create

ssh_keys:
  - name: id_rsa
    type: rsa
    path: ~/.ssh
    comment: johndoe@example.com
    no_passphrase: true
    copy_to_github: true
    set_as_rwr_ssh_key: true
```

## Execution Order

The Bootstrap Process executes the sections in the following order:

1. `packages`
2. `directories`
3. `files`
4. `ssh_keys`
5. `git`
6. `services`
7. `groups`
8. `users`

This order ensures that the necessary dependencies and prerequisites are in place before proceeding with other tasks.

## Conditional Execution

By default, the Bootstrap Process is only executed once during the initial setup. Subsequent runs of RWR will skip the Bootstrap Process unless explicitly specified.

To force the execution of the Bootstrap Process on every run, you can use the `--force-bootstrap` flag:

```bash
rwr all --force-bootstrap
```

This flag will ensure that the Bootstrap Process is executed even if it has been run previously.

## New Features and Enhancements

### Package Management

- The `packages` section now supports the `names` field for installing multiple packages with the same configuration.
- Additional arguments can be specified for package installation using the `args` field.

### File Management

- The `files` section now supports URL sources. RWR will download files from the specified URL before processing them.
- Intelligent renaming is implemented for file operations. If the target path doesn't end with a '/', it's considered a rename operation.

### SSH Key Management

- The `ssh_keys` section has been added to generate and manage SSH keys during the bootstrap process.
- The `set_as_rwr_ssh_key` option allows setting a generated key as the default RWR SSH key for operations.

## Best Practices

When defining your bootstrap file, consider the following best practices:

- Keep the bootstrap file minimal and only include the essential tasks required for the initial setup.
- Use variables and templating to make the bootstrap file more flexible and reusable across different environments.
- Leverage the new features like URL sources and intelligent renaming for more flexible file management.
- Use the `ssh_keys` section to set up necessary SSH keys, including the default RWR SSH key.
- Test the bootstrap file thoroughly to ensure it works as expected on the target systems.
- Document any manual steps or prerequisites that are not covered by the bootstrap file.

By following these best practices, you can create a reliable and maintainable bootstrap process for your RWR-managed systems.

For more information on specific blueprint types and their options, please refer to the respective documentation pages.
