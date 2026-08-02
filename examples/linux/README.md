# Linux Examples

Example blueprint sets for three Linux distributions. Each distribution directory contains the same example in three formats (`yaml/`, `json/`, `toml/`).

## Distributions

- [`Arch/`](./Arch/) — the most extensive example set. Uses `pacman` and `yay` (AUR), with profile-based packages, repositories, scripts, services, users, and more.
- [`Fedora/`](./Fedora/) — uses `dnf`, with RPM Fusion and vendor repositories, systemd services, and profile-based package sets.
- [`Ubuntu/`](./Ubuntu/) — uses `apt`, with GPG-keyed apt repositories, `ufw` scripts, and profile-based package sets.

All three also include small `brew` (Homebrew on Linux) and `cargo` package examples.

## Layout

Each distribution follows the same pattern:

```text
<Distro>/
├── yaml/   # full example in YAML
├── json/   # same content in JSON
└── toml/   # same content in TOML
```

Every format directory has its own README that lists the blueprint types included and how to run them.
