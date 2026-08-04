# Windows Examples (JSON)

A full blueprint set for Windows in JSON format. The sibling `yaml/` and `toml/` directories contain the same content - only the file format differs.

Packages install via `chocolatey` and `winget`; a `scoop` bucket and `cargo` tools are included too.

## Contents

- `init.json` - entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.json` - minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.json` - profile-grouped packages using `chocolatey` and `winget` (base, dev, work, gaming)
- `packages/cargo.json` - Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.json` - Scoop bucket example (`extras`)
- `files/files.json` - copies files from `files/src/` into the home directory
- `fonts/fonts.json` - Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.json` - clones the rwr repository as a sample
- `scripts/scripts.json` - batch scripts (choco/winget updates, WSL setup, RDP config, gaming tweaks, cleanup) per profile
- `services/services.json` - Windows services (`sshd`, `W32Time`)
- `ssh_keys/ssh_keys.json` - ed25519 key generation example
- `users/users.json` - `Developers` group and `builder` user creation
- `configuration/configuration.json` - registry settings via `windows_registry` (show file extensions)

## Running

```bash
# From this directory
rwr all --init-file init.json

# Apply profile-gated entries too
rwr all --init-file init.json --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
