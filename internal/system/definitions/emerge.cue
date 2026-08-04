package providers

providers: "emerge": {
 "name": "emerge",
 "elevated": true,
 "detection": {
  "binary": "emerge",
  "files": [
   "/usr/bin/emerge",
   "/etc/portage",
   "/var/db/repos/gentoo"
  ],
  "distributions": [
   "gentoo"
  ]
 },
 "commands": {
  "install": "-qv",
  "update": "-uDN @world",
  "remove": "-C",
  "list": "qlist -I",
  "listExplicit": "cat /var/lib/portage/world",
  "search": "-s",
  "clean": "--depclean"
 },
 "repository": {
  "paths": {
   "sources": "/etc/portage/repos.conf"
  },
  "add": {
   "steps": [
    {
     "action": "write",
     "dest": "{{ .SourcesPath }}/{{ .Name }}.conf",
     "content": "[{{ .Name }}]\nlocation = {{ .OverlayPath }}\nsync-type = {{ .SyncType }}\nsync-uri = {{ .URL }}\nauto-sync = yes\n"
    },
    {
     "action": "command",
     "exec": "emaint",
     "args": [
      "sync",
      "-r",
      "{{ .Name }}"
     ]
    },
    {
     "action": "command",
     "exec": "emerge",
     "args": [
      "--sync"
     ],
     "condition": "{{ .IsMainRepo }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "remove",
     "path": "{{ .SourcesPath }}/{{ .Name }}.conf"
    },
    {
     "action": "command",
     "exec": "rm",
     "args": [
      "-rf",
      "{{ .OverlayPath }}"
     ]
    }
   ]
  }
 }
}
