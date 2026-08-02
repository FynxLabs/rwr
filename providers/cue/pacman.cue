package providers

providers: "pacman": {
 "name": "pacman",
 "elevated": true,
 "detection": {
  "binary": "pacman",
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
  "install": "-Sy --noconfirm",
  "update": "-Syu --noconfirm",
  "remove": "-R --noconfirm",
  "list": "-Q",
  "search": "-Ss",
  "clean": "-Sc --noconfirm"
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
 }
}
