# Arch Linux Examples

The most extensive example set in the repository. The same content appears in three formats:

- [`yaml/`](./yaml/)
- [`json/`](./json/)
- [`toml/`](./toml/)

## Arch-specific details

- **`pacman`** - system packages, grouped by profile (base, work, dev, gaming, personal, security)
- **`yay`** - AUR packages (`visual-studio-code-bin`, `google-chrome`, and similar)
- **pacman repositories** - multilib, archlinuxcn, chaotic-aur, blackarch, wine-staging, and more, gated by profiles
- **Scripts** - use `pacman -Sy`, multilib enablement, and orphan cleanup (`pacman -Qtdq`)
- Small `brew` (Homebrew on Linux) and `cargo` package sets are included too

See the README inside each format directory for the full file list and run instructions.
