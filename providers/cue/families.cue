// Family templates for the proven clusters. Everything else stays a
// standalone definition — inventing hierarchies for single members obscures
// more than it saves.
package providers

// #PacmanFamily: detection files, distributions, repository handling and core
// packages shared by pacman and every AUR helper. A family-wide fix (the
// staging move, --needed) is one edit here instead of six that must not drift.
#PacmanFamily: #Provider & {
	detection: {
		files: [
 "/etc/pacman.conf",
 "/var/lib/pacman"
]
		distributions: [
 "arch",
 "cachyos",
 "linux/cachyos",
 "manjaro",
 "linux/acreetion"
]
	}
	repository: {
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
	corePackages: {
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
}
}

// #ArchAURHelper: #PacmanFamily plus the clone-and-makepkg install shape.
// Parameterized by the AUR package to clone (#aur — aura ships as aura-bin,
// pamac as pamac-aur) and the package name pacman removes (#removePkg).
#ArchAURHelper: #PacmanFamily & {
	#aur:       string
	#removePkg: string | *#aur
	name:       string
	detection: binary: name
	install: steps: [
		{action: "command", exec: "pacman", args: ["-S", "--needed", "--noconfirm", "base-devel", "git"]},
		{action: "command", exec: "git", args: ["clone", "https://aur.archlinux.org/\(#aur).git", "{{ .TempDir }}/\(name)"]},
		{action: "command", exec: "sh", args: ["-c", "cd {{ .TempDir }}/\(name) && makepkg -si --noconfirm"]},
	]
	remove: steps: [
		{action: "command", exec: "pacman", args: ["-Rns", "--noconfirm", #removePkg]},
	]
}

// #DebianFamily: detection and repository shape shared by the Debian tools
// (apt today; apt-get/aptitude would join here rather than re-declaring it).
#DebianFamily: #Provider & {
	detection: {
		files: [
 "/etc/apt"
]
		distributions: [
 "debian",
 "ubuntu"
]
	}
	repository: {
 "paths": {
  "sources": "/etc/apt/sources.list.d",
  "keys": "/usr/share/keyrings"
 },
 "add": {
  "steps": [
   {
    "action": "download",
    "source": "{{ .KeyURL }}",
    "dest": "{{ .TempKeyPath }}",
    "sha256": "{{ .KeySha256 }}"
   },
   {
    "action": "command",
    "exec": "gpg",
    "args": [
     "--yes",
     "--dearmor",
     "-o",
     "{{ .KeyPath }}",
     "{{ .TempKeyPath }}"
    ]
   },
   {
    "action": "write",
    "dest": "{{ .SourcesPath }}/{{ .Name }}.list",
    "content": "deb [arch={{ .Arch }} signed-by={{ .KeyPath }}] {{ .URL }} {{ .Channel }} {{ .Component }}"
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "remove",
    "path": "{{ .SourcesPath }}/{{ .Name }}.list"
   },
   {
    "action": "remove",
    "path": "{{ .KeyPath }}"
   }
  ]
 }
}
}
