# Arch Linux Examples (YAML)

A full blueprint set for Arch Linux in YAML format. The sibling `json/` and `toml/` directories contain the same content — only the file format differs.

System packages install via `pacman`, AUR packages via `yay`; extra sets use `brew` and `cargo`.

## Contents

- `init.yaml` — entry point: blueprint location, processing order, and the package managers to bootstrap and user-defined variables used by the templates
- `bootstrap.yaml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.yaml` — profile-grouped package sets (base, work, dev, gaming, personal, security) using `pacman` and `yay` (AUR)
- `packages/brew.yaml` — a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.yaml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.yaml` — pacman repositories (multilib, chaotic-aur, blackarch, and more), plus flatpak/snap examples, gated by profiles
- `files/files.yaml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.yaml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.yaml` — profile-gated repository clones (dotfiles, work, dev, gaming, personal)
- `scripts/scripts.yaml` — inline shell scripts (system update via `pacman`, dev setup, gaming tweaks, security hardening)
- `services/services.yaml` — systemd services (`sshd`, `docker`, `postgresql`, `nginx`, and more) per profile
- `ssh_keys/ssh_keys.yaml` — ed25519/RSA key generation per profile, including a GitHub-upload example
- `users/users.yaml` — user/group creation and group membership changes per profile
- `configuration/configuration.yaml` — desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr run --init-file init.yaml

# Apply profile-gated entries too
rwr run --init-file init.yaml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
