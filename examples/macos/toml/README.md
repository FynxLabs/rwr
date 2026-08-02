# macOS Examples (TOML)

A full blueprint set for macOS in TOML format. The sibling `yaml/` and `json/` directories contain the same content — only the file format differs.

Packages install via Homebrew (`brew`); Rust tools via `cargo`.

## Contents

- `init.toml` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.toml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/brew.toml` — profile-grouped Homebrew packages (base, dev, work, gaming)
- `packages/cargo.toml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.toml` — Homebrew tap example (`homebrew/cask-fonts`)
- `files/files.toml` — file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.toml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.toml` — clones the rwr repository as a sample
- `scripts/scripts.toml` — inline shell scripts (Homebrew update, dev setup, Docker Desktop launch)
- `services/services.toml` — launchd service management (Docker)
- `ssh_keys/ssh_keys.toml` — ed25519 key generation example
- `users/users.toml` — `developers` group and `builder` user creation
- `configuration/configuration.toml` — macOS defaults via `macos_defaults` (Dock autohide, Finder hidden files)

## Running

```bash
# From this directory
rwr all --init-file init.toml

# Apply profile-gated entries too
rwr all --init-file init.toml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
