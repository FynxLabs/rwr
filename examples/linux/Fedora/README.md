# Fedora Examples

Example blueprint set for Fedora. The same content appears in three formats:

- [`yaml/`](./yaml/)
- [`json/`](./json/)
- [`toml/`](./toml/)

## Fedora-specific details

- **`dnf`** — system packages, grouped by profile (base, dev, work, gaming, docker, database)
- **dnf repositories** — RPM Fusion, vscode, docker-ce, and steam repositories, gated by profiles
- **Scripts** — use `dnf update` for system updates
- **Services** — systemd units including `firewalld` and `httpd`
- Small `brew` (Homebrew on Linux) and `cargo` package sets are included too

See the README inside each format directory for the full file list and run instructions.
