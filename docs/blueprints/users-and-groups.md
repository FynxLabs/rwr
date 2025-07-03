# Users and Groups Blueprint

The Users and Groups blueprint allows you to manage user accounts and groups on your system. You can create, modify, and remove users, assign them to groups, and set their properties such as password, shell, and home directory.

## Blueprint Structure

The Users and Groups blueprint has the following structure:

```yaml
users:
  - name: john
    action: create
    password: "$6$mysecretpassword"
    groups:
      - users
      - developers
    shell: /bin/bash
    home: /home/john

  - name: jane
    action: modify
    new_name: jane_smith
    new_shell: /bin/zsh
    new_home: /home/jane_smith
    add_groups:
      - designers

  - name: bob
    action: remove
    remove_home: true

groups:
  - name: developers
    action: create

  - name: designers
    action: modify
    new_name: design_team
```

## Blueprint Settings

The following settings are available for the Users and Groups blueprint:

### `users`

An array of user objects representing the user accounts to manage.

#### `name`

The username for the user account.

#### `action`

The action to perform on the user account. Supported actions are `create`, `modify`, and `remove`.

#### `new_name`

The new username for the user account (for the `modify` action).

#### `password`

The encrypted password for the user account (for the `create` action). You can generate an encrypted password using tools like `mkpasswd` or `openssl passwd`.

#### `groups`

An array of group names to which the user should belong (for the `create` action).

#### `add_groups`

An array of group names to which the user should be added (for the `modify` action).

#### `remove_home`

A boolean flag indicating whether the user's home directory should be removed (for the `remove` action).

#### `shell`

The login shell for the user account (for the `create` action).

#### `new_shell`

The new login shell for the user account (for the `modify` action).

#### `home`

The home directory for the user account (for the `create` action).

#### `new_home`

The new home directory for the user account (for the `modify` action).

### Groups Configuration

An array of group objects representing the groups to manage.

#### Group `name`

The name of the group.

#### Group `action`

The action to perform on the group. Supported actions are `create` and `modify`.

#### Group `new_name`

The new name for the group (for the `modify` action).

## Supported Platforms

The Users and Groups blueprint is supported on the following platforms:

- Linux
- macOS

Note that on Windows, creating, modifying, and removing users and groups is not supported by RWR.

## Examples

Here are a few examples of using the Users and Groups blueprint in different formats:

### YAML

```yaml
users:
  - name: alice
    action: create
    password: "$6$secretpassword"
    groups:
      - users
      - admin
    shell: /bin/bash
    home: /home/alice

  - name: bob
    action: modify
    new_name: robert
    new_shell: /bin/zsh
    new_home: /home/robert
    add_groups:
      - developers

  - name: charlie
    action: remove
    remove_home: true

groups:
  - name: admin
    action: create

  - name: developers
    action: modify
    new_name: dev_team
```

### JSON

```json
{
  "users": [
    {
      "name": "david",
      "action": "create",
      "password": "$6$othersecretpassword",
      "groups": [
        "users",
        "staff"
      ],
      "shell": "/bin/zsh",
      "home": "/home/david"
    },
    {
      "name": "eve",
      "action": "modify",
      "new_name": "evelyn",
      "add_groups": [
        "managers"
      ]
    },
    {
      "name": "frank",
      "action": "remove"
    }
  ],
  "groups": [
    {
      "name": "staff",
      "action": "create"
    },
    {
      "name": "managers",
      "action": "modify",
      "new_name": "management"
    }
  ]
}
```

### TOML

```toml
[[users]]
name = "carol"
action = "create"
password = "$6$passwordhash"
groups = ["users", "managers"]
shell = "/bin/fish"
home = "/home/carol"

[[users]]
name = "carol"
action = "modify"
new_name = "carolyn"
new_shell = "/bin/zsh"
new_home = "/home/carolyn"
add_groups = ["designers"]

[[users]]
name = "grace"
action = "remove"
remove_home = true

[[groups]]
name = "managers"
action = "create"

[[groups]]
name = "designers"
action = "modify"
new_name = "design_team"
```

These examples demonstrate how to define users and groups using the Users and Groups blueprint in YAML, JSON, and TOML formats, including the new options for modifying and removing users and groups.

For more information on managing users and groups in RWR, please refer to the [Blueprints Overview](../blueprints-general.md) and the [Best Practices](../best-practices.md) sections of the documentation.
