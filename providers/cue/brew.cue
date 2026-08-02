package providers

providers: "brew": {
 "name": "brew",
 "elevated": false,
 "detection": {
  "binary": "brew",
  "files": [
   "/usr/local/bin/brew",
   "/opt/homebrew/bin/brew",
   "/home/linuxbrew/.linuxbrew/bin/brew",
   "~/.linuxbrew/bin/brew"
  ],
  "distributions": [
   "darwin",
   "linux"
  ]
 },
 "commands": {
  "install": "install -fq",
  "update": "update",
  "remove": "uninstall -fq",
  "list": "list",
  "search": "search",
  "clean": "cleanup -q"
 },
 "install": {
  "steps": [
   {
    "action": "download",
    "source": "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh",
    "dest": "{{ .TempDir }}/brew-install.sh"
   },
   {
    "action": "command",
    "exec": "chmod",
    "args": [
     "+x",
     "{{ .TempDir }}/brew-install.sh"
    ]
   },
   {
    "action": "command",
    "exec": "bash",
    "args": [
     "{{ .TempDir }}/brew-install.sh"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "{{ .TempDir }}/brew-install.sh"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "download",
    "source": "https://raw.githubusercontent.com/Homebrew/install/HEAD/uninstall.sh",
    "dest": "{{ .TempDir }}/brew-uninstall.sh"
   },
   {
    "action": "command",
    "exec": "chmod",
    "args": [
     "+x",
     "{{ .TempDir }}/brew-uninstall.sh"
    ]
   },
   {
    "action": "command",
    "exec": "bash",
    "args": [
     "{{ .TempDir }}/brew-uninstall.sh"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "{{ .TempDir }}/brew-uninstall.sh"
    ]
   }
  ]
 },
 "corePackages": {
  "openssl": [
   "openssl"
  ],
  "build-essentials": [
   "make",
   "cmake",
   "pkg-config",
   "freetype",
   "fontconfig"
  ]
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "brew",
     "args": [
      "tap",
      "{{ .URL }}"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "brew",
     "args": [
      "untap",
      "{{ .URL }}"
     ]
    }
   ]
  }
 }
}
