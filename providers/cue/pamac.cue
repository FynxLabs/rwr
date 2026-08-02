package providers

providers: "pamac": {
 "name": "pamac",
 "elevated": true,
 "detection": {
  "binary": "pamac",
  "files": [
   "/etc/pacman.conf",
   "/var/lib/pacman"
  ],
  "distributions": [
   "arch",
   "cachyos",
   "linux/cachyos",
   "manjaro",
   "linux/acreetion"
  ]
 },
 "commands": {
  "install": "build --no-confirm",
  "update": "upgrade -a --no-confirm",
  "remove": "remove --no-confirm",
  "list": "list -i",
  "search": "search -a",
  "clean": "clean --no-confirm"
 },
 "corePackages": {
  "openssl": [
   "openssl"
  ],
  "build-essentials": [
   "base-devel",
   "cmake",
   "freetype2",
   "fontconfig",
   "pkg-config",
   "libxcb",
   "libxkbcommon",
   "python"
  ]
 },
 "repository": {
  "paths": {
   "sources": "/etc/pacman.d",
   "keys": "/etc/pacman.d/gnupg",
   "config": "/etc/pacman.conf"
  },
  "add": {
   "steps": [
    {
     "action": "append",
     "path": "{{ .ConfigPath }}",
     "content": "[{{ .Name }}]\nServer = {{ .URL }}\n"
    },
    {
     "action": "command",
     "exec": "pacman-key",
     "args": [
      "--recv-keys",
      "{{ .KeyID }}"
     ],
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "pacman-key",
     "args": [
      "--lsign-key",
      "{{ .KeyID }}"
     ],
     "condition": "{{ .HasKey }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "pacman-key",
     "args": [
      "--delete",
      "{{ .KeyID }}"
     ],
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "remove_section",
     "path": "{{ .ConfigPath }}",
     "section": "{{ .Name }}"
    }
   ]
  }
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "pacman",
    "args": [
     "-S",
     "--needed",
     "--noconfirm",
     "base-devel",
     "git"
    ]
   },
   {
    "action": "command",
    "exec": "git",
    "args": [
     "clone",
     "https://aur.archlinux.org/pamac-aur.git",
     "{{ .TempDir }}/pamac"
    ]
   },
   {
    "action": "command",
    "exec": "sh",
    "args": [
     "-c",
     "cd {{ .TempDir }}/pamac && makepkg -si --noconfirm"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "pacman",
    "args": [
     "-Rns",
     "--noconfirm",
     "pamac-aur"
    ]
   }
  ]
 }
}
