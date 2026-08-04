# Windows Examples (TOML)

A full blueprint set for Windows in TOML format. The sibling `yaml/` and `json/` directories contain the same content - only the file format differs.

Packages install via `chocolatey` and `winget`; a `scoop` bucket and `cargo` tools are included too.

## Contents

- `init.toml` - entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.toml` - minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.toml` - profile-grouped packages using `chocolatey` and `winget` (base, dev, work, gaming)
- `packages/cargo.toml` - Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.toml` - Scoop bucket example (`extras`)
- `files/files.toml` - copies files from `files/src/` into the home directory
- `fonts/fonts.toml` - Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.toml` - clones the rwr repository as a sample
- `scripts/scripts.toml` - batch scripts (choco/winget updates, WSL setup, RDP config, gaming tweaks, cleanup) per profile
- `services/services.toml` - Windows services (`sshd`, `W32Time`)
- `ssh_keys/ssh_keys.toml` - ed25519 key generation example
- `users/users.toml` - `Developers` group and `builder` user creation
- `configuration/configuration.toml` - registry settings via `windows_registry` (show file extensions)

## Running

```bash
# From this directory
rwr all --init-file init.toml

# Apply profile-gated entries too
rwr all --init-file init.toml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
