# macOS Examples (YAML)

A full blueprint set for macOS in YAML format. The sibling `json/` and `toml/` directories contain the same content — only the file format differs.

Packages install via Homebrew (`brew`); Rust tools via `cargo`.

## Contents

- `init.yaml` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.yaml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/brew.yaml` — profile-grouped Homebrew packages (base, dev, work, gaming)
- `packages/cargo.yaml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.yaml` — Homebrew tap example (`homebrew/cask-fonts`)
- `files/files.yaml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.yaml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.yaml` — clones the rwr repository as a sample
- `scripts/scripts.yaml` — inline shell scripts (Homebrew update, dev setup, Docker Desktop launch)
- `services/services.yaml` — launchd service management (Docker)
- `ssh_keys/ssh_keys.yaml` — ed25519 key generation example
- `users/users.yaml` — `developers` group and `builder` user creation
- `configuration/configuration.yaml` — macOS defaults via `macos_defaults` (Dock autohide, Finder hidden files)

## Running

```bash
# From this directory
rwr run --init-file init.yaml

# Apply profile-gated entries too
rwr run --init-file init.yaml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
