# Arch Linux Examples (TOML)

A full blueprint set for Arch Linux in TOML format. The sibling `yaml/` and `json/` directories contain the same content — only the file format differs.

System packages install via `pacman`, AUR packages via `yay`; extra sets use `brew` and `cargo`.

## Contents

- `init.toml` — entry point: blueprint location, processing order, and the package managers to bootstrap and user-defined variables used by the templates
- `bootstrap.toml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.toml` — profile-grouped package sets (base, work, dev, gaming, personal, security) using `pacman` and `yay` (AUR)
- `packages/brew.toml` — a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.toml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.toml` — pacman repositories (multilib, chaotic-aur, blackarch, and more), plus flatpak/snap examples, gated by profiles
- `files/files.toml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.toml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.toml` — profile-gated repository clones (dotfiles, work, dev, gaming, personal)
- `scripts/scripts.toml` — inline shell scripts (system update via `pacman`, dev setup, gaming tweaks, security hardening)
- `services/services.toml` — systemd services (`sshd`, `docker`, `postgresql`, `nginx`, and more) per profile
- `ssh_keys/ssh_keys.toml` — ed25519/RSA key generation per profile, including a GitHub-upload example
- `users/users.toml` — user/group creation and group membership changes per profile
- `configuration/configuration.toml` — desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr run --init-file init.toml

# Apply profile-gated entries too
rwr run --init-file init.toml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
