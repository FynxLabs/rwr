# Ubuntu Examples (TOML)

A full blueprint set for Ubuntu in TOML format. The sibling `yaml/` and `json/` directories contain the same content - only the file format differs.

System packages install via `apt`; extra sets use `brew` and `cargo`.

## Contents

- `init.toml` - entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.toml` - minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.toml` - profile-grouped package sets (base, dev, work, gaming, docker, database) using `apt`
- `packages/brew.toml` - a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.toml` - Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.toml` - apt repositories with GPG keys (docker, hashicorp)
- `files/files.toml` - copies files from `files/src/` into the home directory
- `fonts/fonts.toml` - Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.toml` - clones the rwr repository as a sample
- `scripts/scripts.toml` - inline shell scripts (system update via `apt`, firewall, dev/server/desktop setup per profile)
- `services/services.toml` - systemd services (`ssh`, `docker`, `ufw`)
- `ssh_keys/ssh_keys.toml` - ed25519 key generation example
- `users/users.toml` - group creation plus `builder` and `deploy` user examples
- `configuration/configuration.toml` - desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr all --init-file init.toml

# Apply profile-gated entries too
rwr all --init-file init.toml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
