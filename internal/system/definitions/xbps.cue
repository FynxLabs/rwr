package providers

providers: "xbps": {
 "name": "xbps",
 "elevated": true,
 "detection": {
  "binary": "xbps-install",
  "files": [
   "/usr/bin/xbps-install",
   "/usr/share/xbps.d",
   "/var/db/xbps"
  ],
  "distributions": [
   "void"
  ]
 },
 "commands": {
  "install": "-Sy",
  "update": "-Su",
  "remove": "-R",
  "list": "xbps-query -l",
  "listExplicit": "xbps-query -m",
  "search": "xbps-query -Rs",
  "clean": "xbps-remove -O"
 },
 "repository": {
  "paths": {
   "sources": "/etc/xbps.d",
   "keys": "/var/db/xbps/keys"
  },
  "add": {
   "steps": [
    {
     "action": "write",
     "dest": "{{ .SourcesPath }}/{{ .Name }}.conf",
     "content": "repository={{ .URL }}\n"
    },
    {
     "action": "download",
     "source": "{{ .KeyURL }}",
     "dest": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "xbps-install",
     "args": [
      "-S"
     ]
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
     "action": "remove",
     "path": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "xbps-install",
     "args": [
      "-S"
     ]
    }
   ]
  }
 }
}
