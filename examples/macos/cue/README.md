# macOS Examples (JSON)

A full blueprint set for macOS in JSON format. The sibling `yaml/` and `toml/` directories contain the same content - only the file format differs.

Packages install via Homebrew (`brew`); Rust tools via `cargo`.

## Contents

- `init.json` - entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.json` - minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/brew.json` - profile-grouped Homebrew packages (base, dev, work, gaming)
- `packages/cargo.json` - Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.json` - Homebrew tap example (`homebrew/cask-fonts`)
- `files/files.json` - file, directory, and template management (sources live in `files/src/`)
- `fonts/fonts.json` - Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.json` - clones the rwr repository as a sample
- `scripts/scripts.json` - inline shell scripts (Homebrew update, dev setup, Docker Desktop launch)
- `services/services.json` - launchd service management (Docker)
- `ssh_keys/ssh_keys.json` - ed25519 key generation example
- `users/users.json` - `developers` group and `builder` user creation
- `configuration/configuration.json` - macOS defaults via `macos_defaults` (Dock autohide, Finder hidden files)

## Running

```bash
# From this directory
rwr all --init-file init.json

# Apply profile-gated entries too
rwr all --init-file init.json --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
