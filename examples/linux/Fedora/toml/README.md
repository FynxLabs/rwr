# Fedora Examples (TOML)

A full blueprint set for Fedora in TOML format. The sibling `yaml/` and `json/` directories contain the same content — only the file format differs.

System packages install via `dnf`; extra sets use `brew` and `cargo`.

## Contents

- `init.toml` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.toml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.toml` — profile-grouped package sets (base, dev, work, gaming, docker, database) using `dnf`
- `packages/brew.toml` — a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.toml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.toml` — dnf repositories (RPM Fusion, vscode, docker-ce, steam) gated by profiles
- `files/files.toml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.toml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.toml` — clones the rwr repository as a sample
- `scripts/scripts.toml` — inline shell scripts (system update via `dnf`, dev setup, docker setup)
- `services/services.toml` — systemd services (`sshd`, `firewalld`, `docker`, `postgresql`, `httpd`) per profile
- `ssh_keys/ssh_keys.toml` — ed25519 key generation example
- `users/users.toml` — group creation and adding the current user to `docker`
- `configuration/configuration.toml` — desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr run --init-file init.toml

# Apply profile-gated entries too
rwr run --init-file init.toml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
