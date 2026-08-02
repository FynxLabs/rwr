# Users and Groups Blueprint

With the Users and Groups blueprint, you manage user accounts and groups on your system. You can create, modify, and remove users, assign them to groups, and set their properties such as password, shell, and home directory.

## Blueprint Structure

The Users and Groups blueprint has the following structure:

```yaml
users:
  - name: john
    action: create
    password: "$6$rounds=656000$saltsalt$hashhashhash"
    uid: "1500"
    comment: "John Doe"
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
    remove_groups:
      - interns
    unlock: true

  - name: bob
    action: remove
    remove_home: true

groups:
  - name: developers
    action: create
    gid: "3000"

  - name: designers
    action: modify
    new_name: design_team
```

See [Fields Common to Every Blueprint](common-fields.md) for `profiles`,
`import`, `interactive`, and the rule that an unknown key is an error.

## Actions

`create`, `modify` and `remove` for both users and groups. `delete` is an
accepted alias for `remove`.

`create` is idempotent. If the account or group already exists, RWR does not
fail. It converges the existing one to the attributes that the entry declares
(`usermod` on Linux, `dscl -create` on macOS). If a password is given, RWR sets
it. RWR treats a declared `home` on an existing account as the intended home,
not as a request to relocate the current one. To relocate, use `modify` with
`new_home`.

`remove` on an account or group that does not exist succeeds and does nothing.

## `users` settings

| Setting | Description |
|---------|-------------|
| `name` | The username (required if `import` is not provided) |
| `action` | `create`, `modify`, `remove` (or its alias `delete`) |
| `uid` | User ID to assign. It is a **string**: write `uid: "1500"` |
| `password` | See [Passwords](#passwords) |
| `groups` | Supplementary groups to put the user in (`create`) |
| `add_groups` | Groups to add the user to (`modify`) |
| `remove_groups` | Groups to remove the user from (`modify`) |
| `shell` | Login shell (`create`) |
| `new_shell` | New login shell (`modify`) |
| `home` | Home directory (`create`) |
| `new_home` | New home directory (`modify`) |
| `comment` | The GECOS/real-name field |
| `system` | Create as a system account |
| `expire` | Account expiration date, `YYYY-MM-DD` |
| `lock` | Lock the account (`modify`) |
| `unlock` | Unlock the account (`modify`) |
| `remove_home` | Remove the home directory too (`remove`) |
| `new_name` | Rename the account (`modify`) |
| `profiles`, `import`, `interactive` | See [common fields](common-fields.md) |

## `groups` settings

| Setting | Description |
|---------|-------------|
| `name` | The group name (required if `import` is not provided) |
| `action` | `create`, `modify`, `remove` (or its alias `delete`) |
| `gid` | Group ID to assign. It is a **string**: write `gid: "3000"` |
| `system` | Create as a system group |
| `new_name` | Rename the group (`modify`) |
| `profiles`, `import` | See [common fields](common-fields.md) |

Groups have no `interactive` field.

## Passwords

**On Linux**, RWR hands the password to `chpasswd` on standard input, never as a
command argument. Thus it does not appear in `ps` or in rwr's debug output. The
value can be either:

- **Cleartext**, which `chpasswd` hashes itself, or
- **A crypt(3) hash** — `$6$…`, `$y$…`, a 13-character DES hash, or one of the
  locked markers `!`, `!!`, `*`, `*LK*` — which RWR passes through with
  `chpasswd -e`.

RWR detects which of the two you wrote. There is no field to declare it.

> [!WARNING]
> **On macOS**, RWR passes the password to `dscl . -passwd` as a command-line
> argument. macOS computes its own salted blob, and there is no hash to
> pre-compute. The password is therefore briefly visible in `ps` to every local user on
> the machine, and lands in sudo's syslog record. RWR logs a warning each time
> it does this. Prefer setting macOS passwords out of band.

Both `create` and `modify` apply a `password`.

## Supported Platforms

| Platform | Support |
|----------|---------|
| Linux | Full, via shadow-utils: `useradd`, `usermod`, `userdel`, `groupadd`, `groupmod`, `groupdel`, `gpasswd`, `chpasswd` |
| macOS | Full, via Open Directory: `sysadminctl` when present, otherwise `dscl`, plus `dseditgroup`, `createhomedir` and `pwpolicy` |
| Windows | Not implemented. RWR logs a warning for each entry and skips it |

### What differs on macOS

macOS is supported. Open Directory has no equivalent for some of the fields.
RWR warns and does not report a false success:

| Field | On macOS |
|-------|----------|
| `system` | **Ignored, with a warning.** UIDs and GIDs below 501 are reserved by Apple and RWR does not allocate them |
| `expire` | **Ignored, with a warning.** A local Open Directory account has no expiration field |
| `new_home` | RWR rewrites the `NFSHomeDirectory` record, but it does **not move the contents** of the directory. RWR warns when it does this |
| `remove_home` | Honored through `sysadminctl -deleteUser` (which is also why RWR passes `-keepHome` when the flag is false). If `sysadminctl` is not available, the `dscl` fallback deletes the record only and warns that the home directory was left in place |
| `lock` / `unlock` | Applied with `pwpolicy -disableuser` / `-enableuser` |
| primary group | New accounts land in `staff` (GID 20). There is no field to choose another |
| `uid` / `gid` | Allocated from 501 upwards when not declared |

Other fields map straight across: `shell` to `UserShell`, `comment` to
`RealName`, `home` to `NFSHomeDirectory`, `groups`/`add_groups`/`remove_groups`
to `dseditgroup`, `new_name` to a `RecordName` change (applied last, after every
other edit).

## Examples

Examples in YAML, JSON, and TOML:

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

These examples define users and groups in YAML, JSON, and TOML.

## Blueprint Imports

Import user and group definitions from other files:

```yaml
users:
  # Import shared user accounts
  - import: ../../Common/users/base-users.yaml

  # Add environment-specific users
  - name: dev_user
    action: create
    password: "$6$secrethash"
    groups:
      - developers
    shell: /bin/bash
    profiles:
      - dev

groups:
  # Import shared groups
  - import: ../../Common/users/base-groups.yaml

  # Add local groups
  - name: local_admins
    action: create
```

You can then keep common user and group configurations separate from environment-specific ones.

For more information, see the [Blueprints Overview](../blueprints-general.md) and [Best Practices](../best-practices.md).
