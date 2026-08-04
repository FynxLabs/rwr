package providers

providers: "flatpak": {
 "name": "flatpak",
 "elevated": false,
 "detection": {
  "binary": "flatpak",
  "files": [
   "/usr/bin/flatpak",
   "~/.local/share/flatpak",
   "/var/lib/flatpak"
  ],
  "distributions": [
   "linux"
  ]
 },
 "commands": {
  "install": "install -y",
  "update": "update -y",
  "remove": "uninstall -y",
  "list": "list",
  "listExplicit": "list --app --columns=application",
  "search": "search",
  "clean": "uninstall --unused -y"
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-add",
      "--if-not-exists",
      "{{ .Name }}",
      "{{ .URL }}"
     ]
    },
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-add",
      "--if-not-exists",
      "--user",
      "{{ .Name }}",
      "{{ .URL }}"
     ],
     "condition": "{{ .UserMode }}"
    },
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-add",
      "--if-not-exists",
      "--system",
      "{{ .Name }}",
      "{{ .URL }}"
     ],
     "condition": "{{ not .UserMode }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-delete",
      "{{ .Name }}"
     ]
    },
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-delete",
      "--user",
      "{{ .Name }}"
     ],
     "condition": "{{ .UserMode }}"
    },
    {
     "action": "command",
     "exec": "flatpak",
     "args": [
      "remote-delete",
      "--system",
      "{{ .Name }}"
     ],
     "condition": "{{ not .UserMode }}"
    }
   ]
  }
 }
}
