# Ubuntu Examples

Example blueprint set for Ubuntu. The same content appears in three formats:

- [`yaml/`](./yaml/)
- [`json/`](./json/)
- [`toml/`](./toml/)

## Ubuntu-specific details

- **`apt`** - system packages, grouped by profile (base, dev, work, gaming, docker, database)
- **apt repositories** - docker and hashicorp repositories with GPG `key_url`, `channel`, `component`, and `arch` fields
- **Scripts** - use `apt update && apt upgrade`, plus `ufw` firewall setup
- **Services** - systemd units (`ssh`, `docker`, `ufw`)
- Small `brew` (Homebrew on Linux) and `cargo` package sets are included too

See the README inside each format directory for the full file list and run instructions.
