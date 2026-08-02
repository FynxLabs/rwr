# macOS Examples

Example blueprint set for macOS, provided in three formats with identical content:

- [`yaml/`](./yaml/)
- [`json/`](./json/)
- [`toml/`](./toml/)

## What the examples use

- **Homebrew (`brew`)** — the primary package manager for all packages, plus a tap example under `repositories/`
- **`cargo`** — Rust CLI tools
- **`macos_defaults`** — system configuration via `defaults` (Dock, Finder)
- **launchd services** — Docker service management under `services/`

Each format directory has its own README that lists every blueprint type included and how to run the example.
