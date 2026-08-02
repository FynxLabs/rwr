# Windows Examples (YAML)

A full blueprint set for Windows in YAML format. The sibling `json/` and `toml/` directories contain the same content — only the file format differs.

Packages install via `chocolatey` and `winget`; a `scoop` bucket and `cargo` tools are included too.

## Contents

- `init.yaml` — entry point: blueprint location, processing order, and the package managers to bootstrap
- `bootstrap.yaml` — minimal first-run setup: `git` + `curl`, a `~/git` directory, and a GitHub SSH key
- `packages/packages.yaml` — profile-grouped packages using `chocolatey` and `winget` (base, dev, work, gaming)
- `packages/cargo.yaml` — Rust tools via `cargo` (`bat`, `bottom`)
- `repositories/repositories.yaml` — Scoop bucket example (`extras`)
- `files/files.yaml` — copies files from `files/src/` into the home directory
- `fonts/fonts.yaml` — Nerd Font installs (FiraCode, JetBrainsMono, Hack, SourceCodePro)
- `git/git.yaml` — clones the rwr repository as a sample
- `scripts/scripts.yaml` — batch scripts (choco/winget updates, WSL setup, RDP config, gaming tweaks, cleanup) per profile
- `services/services.yaml` — Windows services (`sshd`, `W32Time`)
- `ssh_keys/ssh_keys.yaml` — ed25519 key generation example
- `users/users.yaml` — `Developers` group and `builder` user creation
- `configuration/configuration.yaml` — registry settings via `windows_registry` (show file extensions)

## Running

```bash
# From this directory
rwr all --init-file init.yaml

# Apply profile-gated entries too
rwr all --init-file init.yaml --profile dev
```

Entries without a `profiles` field always apply. Entries with `profiles` only apply when you pass a matching `--profile`.
