package providers

providers: "paru": {
 "name": "paru",
 "elevated": false,
 "detection": {
  "binary": "paru",
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
  "install": "-S --noconfirm",
  "update": "-Sua --noconfirm",
  "remove": "-Rns --noconfirm",
  "list": "-Qm",
  "search": "-Ss",
  "clean": "-Scc --noconfirm"
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
     "https://aur.archlinux.org/paru.git",
     "{{ .TempDir }}/paru"
    ]
   },
   {
    "action": "command",
    "exec": "sh",
    "args": [
     "-c",
     "cd {{ .TempDir }}/paru && makepkg -si --noconfirm"
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
     "paru"
    ]
   }
  ]
 }
}
