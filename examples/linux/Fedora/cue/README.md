# Fedora Examples (JSON)

A full blueprint set for Fedora in JSON format. The sibling `yaml/` and `toml/` directories contain the same content — only the file format differs.

System packages install via `dnf`; extra sets use `brew` and `cargo`.

## Contents

- `init.json` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.json` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.json` — profile-grouped package sets (base, dev, work, gaming, docker, database) using `dnf`
- `packages/brew.json` — a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.json` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.json` — dnf repositories (RPM Fusion, vscode, docker-ce, steam) gated by profiles
- `files/files.json` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.json` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.json` — clones the rwr repository as a sample
- `scripts/scripts.json` — inline shell scripts (system update via `dnf`, dev setup, docker setup)
- `services/services.json` — systemd services (`sshd`, `firewalld`, `docker`, `postgresql`, `httpd`) per profile
- `ssh_keys/ssh_keys.json` — ed25519 key generation example
- `users/users.json` — group creation and adding the current user to `docker`
- `configuration/configuration.json` — desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr all --init-file init.json

# Apply profile-gated entries too
rwr all --init-file init.json --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
