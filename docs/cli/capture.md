# rwr capture

Turn a handcrafted machine into a blueprint tree. The scan finds what you
put there — explicitly-installed packages per manager (`pacman -Qe`,
`brew leaves`, `cargo install --list`, …, so dependency noise never
appears), dotfiles and `~/.config` entries minus known cache/state noise,
enabled services, git checkouts — and a per-category form decides what to
keep.

```bash
# Interactive: one selection page per category
rwr capture ~/git/me/rwr-blueprints

# Take the defaults without the form (CI/scripting)
rwr capture --all out/

# Choose the format and add a root manifest matched to this machine
rwr capture --format cue --manifest out/
```

Defaults: explicit packages and known dotfiles pre-selected, services off
(most enabled units are distro plumbing), git checkouts on. Secret-shaped
paths (ssh, gnupg, netrc, credential dirs) are never pre-selected and warn
if chosen. Selected configs are copied under `files/src/` with entries
targeting their origin via `{{ .User.home }}`.

The generated tree is validated before capture reports success — output
that needs hand-repair is a failure, not a capture. From there:
`rwr all --init-file <dir>` provisions the next machine.
