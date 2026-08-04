# Ubuntu Examples (JSON)

A full blueprint set for Ubuntu in JSON format. The sibling `yaml/` and `toml/` directories contain the same content - only the file format differs.

System packages install via `apt`; extra sets use `brew` and `cargo`.

## Contents

- `init.json` - entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.json` - minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.json` - profile-grouped package sets (base, dev, work, gaming, docker, database) using `apt`
- `packages/brew.json` - a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.json` - Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.json` - apt repositories with GPG keys (docker, hashicorp)
- `files/files.json` - copies files from `files/src/` into the home directory
- `fonts/fonts.json` - Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.json` - clones the rwr repository as a sample
- `scripts/scripts.json` - inline shell scripts (system update via `apt`, firewall, dev/server/desktop setup per profile)
- `services/services.json` - systemd services (`ssh`, `docker`, `ufw`)
- `ssh_keys/ssh_keys.json` - ed25519 key generation example
- `users/users.json` - group creation plus `builder` and `deploy` user examples
- `configuration/configuration.json` - desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr all --init-file init.json

# Apply profile-gated entries too
rwr all --init-file init.json --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
