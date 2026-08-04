package providers

providers: "eopkg": {
 "name": "eopkg",
 "elevated": true,
 "detection": {
  "binary": "eopkg",
  "files": [
   "/usr/bin/eopkg",
   "/var/lib/eopkg",
   "/etc/eopkg"
  ],
  "distributions": [
   "solus"
  ]
 },
 "commands": {
  "install": "it -y",
  "update": "ur",
  "remove": "rm -y",
  "list": "li",
  "search": "sr",
  "clean": "rmo -y"
 },
 "repository": {
  "paths": {
   "keys": "/etc/eopkg/keys"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "eopkg",
     "args": [
      "add-repo",
      "{{ .Name }}",
      "{{ .URL }}"
     ]
    },
    {
     "action": "download",
     "source": "{{ .KeyURL }}",
     "dest": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "eopkg",
     "args": [
      "import",
      "{{ .KeyPath }}"
     ],
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "eopkg",
     "args": [
      "update-repo",
      "{{ .Name }}"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "eopkg",
     "args": [
      "remove-repo",
      "{{ .Name }}"
     ]
    },
    {
     "action": "remove",
     "path": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "command",
     "exec": "eopkg",
     "args": [
      "update-repo",
      "--all"
     ]
    }
   ]
  }
 }
}
