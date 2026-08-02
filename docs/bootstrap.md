# Bootstrap Process

The bootstrap process in Rinse, Wash, Repeat (RWR) sets the initial system configuration. It puts the prerequisites in place before RWR runs the main blueprints. This page explains how the bootstrap process works and how to define the bootstrap file.

## Overview

RWR runs the bootstrap process before the other blueprints. It typically includes tasks such as:

- Installing essential packages and tools
- Setting up package managers
- Creating required directories and files
- Configuring system settings and permissions
- Setting up SSH keys

You define the bootstrap process in a separate blueprint file named `bootstrap.yaml` (or `bootstrap.json` or `bootstrap.toml`, in the format that you selected).

## Bootstrap File Structure

The structure of the bootstrap file is similar to other blueprint files in RWR. It can include the following sections:

- `packages`: Defines the packages that RWR installs during bootstrap.
- `files`: Defines the files that RWR creates or modifies during bootstrap.
- `directories`: Defines the directories that RWR creates during bootstrap.
- `git`: Defines the Git repositories that RWR clones during bootstrap.
- `services`: Defines the services that RWR manages (starts, stops, enables, disables) during bootstrap.
- `users`: Defines the user accounts that RWR creates during bootstrap.
- `groups`: Defines the groups that RWR creates during bootstrap.
- `ssh_keys`: Defines the SSH keys that RWR generates during bootstrap.

Here is an example of a `bootstrap.yaml` file:

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

The bootstrap process runs the sections in this order:

1. `packages`
2. `directories`
3. `files`
4. `ssh_keys`
5. `git`
6. `services`
7. `groups`
8. `users`

This order puts the dependencies in place before the tasks that need them.

## Conditional Execution

By default, RWR runs the bootstrap process one time, during the initial setup. Later runs skip it.

To run the bootstrap process on every run, use the `--force-bootstrap` flag:

```bash
rwr all --force-bootstrap
```

With this flag, RWR runs the bootstrap process again on each run.

## New Features and Enhancements

### Package Management

- The `packages` section now supports the `names` field. You can install many packages with the same configuration.
- You can give additional arguments for package installation with the `args` field.

### File Management

- The `files` section now supports URL sources. RWR downloads the file from the URL before it processes the file.
- If the target path does not end with '/', RWR renames the file.

### SSH Key Management

- The `ssh_keys` section generates and manages SSH keys during bootstrap.
- With the `set_as_rwr_ssh_key` option, RWR uses the generated key as its default SSH key.

## Best Practices

When you define your bootstrap file, use these practices:

- Keep the bootstrap file minimal. Include only the essential tasks for the initial setup.
- Use variables and templates to make the bootstrap file flexible and reusable across different environments.
- Use URL sources and renaming for file management.
- Use the `ssh_keys` section to generate the necessary SSH keys, with the default RWR SSH key.
- Test the bootstrap file on the target systems. Make sure that it does what you expect.
- Document the manual steps or prerequisites that the bootstrap file does not cover.

For more information on the blueprint types and their settings, read the documentation page for each type.
