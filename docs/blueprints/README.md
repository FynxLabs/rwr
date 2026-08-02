# Blueprint Type Docs

One page per blueprint type.

Read [Fields Common to Every Blueprint](common-fields.md) first — it covers `profiles`, `import`, `interactive`, and the rule that an unknown key is an error.

- [Packages](packages.md) — installs and removes packages across many package managers.
- [Repositories](repositories.md) — adds and removes package manager repositories.
- [Files](files.md) — copies, moves, deletes, creates, and modifies files; renders templates.
- [Directories](directories.md) — creates, deletes, copies, and moves directories; sets permissions and owner.
- [Configuration](configuration.md) — manages system configuration: dconf/gsettings (Linux), defaults (macOS), the registry (Windows).
- [Services](services.md) — starts, stops, enables, and disables services.
- [Git](git.md) — clones and manages Git repositories.
- [Scripts](scripts.md) — runs custom scripts for tasks other blueprint types do not cover.
- [SSH Keys](ssh-keys.md) — generates and manages SSH keys; can upload public keys to GitHub.
- [Users and Groups](users-and-groups.md) — creates, modifies, and removes user accounts and groups.
- [Fonts](fonts.md) — installs and removes Nerd Fonts.
