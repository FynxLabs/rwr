package providers

providers: "gnome-extensions": {
 "name": "gnome-extensions",
 "elevated": false,
 "detection": {
  "binary": "gnome-extensions",
  "files": [
   "/usr/bin/gnome-extensions",
   "/usr/bin/gext",
   "~/.local/share/gnome-shell/extensions"
  ],
  "distributions": [
   "linux"
  ]
 },
 "commands": {
  "install": "install",
  "update": "update",
  "remove": "uninstall",
  "list": "list --user --enabled",
  "search": "search",
  "clean": ""
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "install",
      "{{ .Path }}"
     ],
     "condition": "{{ .IsLocalFile }}"
    },
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "install",
      "--force",
      "{{ .ExtensionID }}"
     ],
     "condition": "{{ not .IsLocalFile }}"
    },
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "enable",
      "{{ .UUID }}"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "disable",
      "{{ .UUID }}"
     ]
    },
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "uninstall",
      "{{ .UUID }}"
     ]
    },
    {
     "action": "command",
     "exec": "gnome-extensions",
     "args": [
      "reset",
      "{{ .UUID }}"
     ],
     "condition": "{{ .ResetSettings }}"
    }
   ]
  }
 }
}
