package providers

providers: "slackpkg": {
 "name": "slackpkg",
 "elevated": true,
 "detection": {
  "binary": "slackpkg",
  "files": [
   "/usr/sbin/slackpkg",
   "/etc/slackpkg",
   "/var/lib/slackpkg"
  ],
  "distributions": [
   "slackware"
  ]
 },
 "commands": {
  "install": "install",
  "update": "update",
  "remove": "remove",
  "list": "search installed",
  "search": "search",
  "clean": "clean-system"
 },
 "repository": {
  "paths": {
   "sources": "/etc/slackpkg/mirrors",
   "keys": "/etc/slackpkg/gpg",
   "config": "/etc/slackpkg/blacklist"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "sed",
     "args": [
      "-i",
      "s/^[^#].*//",
      "{{ .SourcesPath }}"
     ]
    },
    {
     "action": "append",
     "path": "{{ .SourcesPath }}",
     "content": "{{ .URL }}"
    },
    {
     "action": "download",
     "source": "{{ .KeyURL }}",
     "dest": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "slackpkg",
     "args": [
      "update",
      "gpg"
     ],
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "slackpkg",
     "args": [
      "update"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "remove_line",
     "path": "{{ .SourcesPath }}",
     "match": "{{ .URL }}"
    },
    {
     "action": "remove",
     "path": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "slackpkg",
     "args": [
      "update"
     ]
    }
   ]
  }
 }
}
