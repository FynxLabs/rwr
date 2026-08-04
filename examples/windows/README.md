# Windows Examples

Example blueprint set for Windows, provided in three formats with identical content:

- [`yaml/`](./yaml/)
- [`json/`](./json/)
- [`toml/`](./toml/)

## What the examples use

- **Chocolatey (`chocolatey`)** - primary package manager for most packages
- **`winget`** - Microsoft Store / winget packages (Windows Terminal, PowerShell)
- **`scoop`** - bucket example under `repositories/`
- **`cargo`** - Rust CLI tools
- **`windows_registry`** - system configuration via the registry (show file extensions)
- **Batch scripts** - WSL setup, RDP configuration, gaming tweaks, and cleanup under `scripts/`

Each format directory has its own README that lists every blueprint type included and how to run the example.
