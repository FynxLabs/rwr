# Fedora Examples (YAML)

A full blueprint set for Fedora in YAML format. The sibling `json/` and `toml/` directories contain the same content — only the file format differs.

System packages install via `dnf`; extra sets use `brew` and `cargo`.

## Contents

- `init.yaml` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.yaml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.yaml` — profile-grouped package sets (base, dev, work, gaming, docker, database) using `dnf`
- `packages/brew.yaml` — a small Homebrew-on-Linux set (`neovim`, `jq`)
- `packages/cargo.yaml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.yaml` — dnf repositories (RPM Fusion, vscode, docker-ce, steam) gated by profiles
- `files/files.yaml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.yaml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.yaml` — clones the rwr repository as a sample
- `scripts/scripts.yaml` — inline shell scripts (system update via `dnf`, dev setup, docker setup)
- `services/services.yaml` — systemd services (`sshd`, `firewalld`, `docker`, `postgresql`, `httpd`) per profile
- `ssh_keys/ssh_keys.yaml` — ed25519 key generation example
- `users/users.yaml` — group creation and adding the current user to `docker`
- `configuration/configuration.yaml` — desktop settings via `gsettings` and `dconf`

## Running

```bash
# From this directory
rwr all --init-file init.yaml

# Apply profile-gated entries too
rwr all --init-file init.yaml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
