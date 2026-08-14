package providers

providers: "brew": {
 "name": "brew",
 "elevated": false,
    // brew refuses to run as root, but a cask install shells out to sudo to
    // write into /Applications. Without warming the credential cache first,
    // that prompt lands on /dev/tty behind the dashboard and the run hangs.
    "escalates": true,
 // Homebrew 6 made "ask mode" the default: every install stops at a
 // "Do you want to proceed? [y/n]" prompt, which deadlocks an automated
 // run. HOMEBREW_NO_ASK restores unattended installs (older brew ignores
 // it; a -y flag would be rejected by older brew as an unknown option).
 // NONINTERACTIVE does the same for Homebrew's official install.sh used
 // by the bootstrap steps below (it skips the "Press RETURN" prompt).
 "environment": {
  "HOMEBREW_NO_ASK": "1",
  "NONINTERACTIVE": "1"
 },
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
  "listExplicit": "leaves",
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
    },
    // Homebrew 6 refuses to load formulae/casks from untrusted third-party
    // taps until `brew trust` records them. Declaring the tap in a blueprint
    // IS the operator's trust decision, so trust follows the tap. Optional:
    // older brew has no trust command and must not fail the add.
    {
     "action": "command",
     "exec": "brew",
     "args": [
      "trust",
      "--tap",
      "{{ .URL }}"
     ],
     "optional": true
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
