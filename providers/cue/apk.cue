package providers

providers: "apk": {
 "name": "apk",
 "elevated": true,
 "detection": {
  "binary": "apk",
  "files": [
   "/sbin/apk",
   "/etc/apk",
   "/var/cache/apk"
  ],
  "distributions": [
   "alpine"
  ]
 },
 "commands": {
  "install": "add",
  "update": "update",
  "remove": "del",
  "list": "info",
  "listExplicit": "cat /etc/apk/world",
  "search": "search",
  "clean": "cache clean"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "openssl-dev"
  ],
  "build-essentials": [
   "build-base",
   "cmake",
   "pkgconfig",
   "freetype-dev",
   "fontconfig-dev",
   "libxcb-dev",
   "libxkbcommon-dev",
   "python3"
  ]
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "apk",
    "args": [
     "update"
    ]
   },
   {
    "action": "command",
    "exec": "apk",
    "args": [
     "upgrade"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "apk",
    "args": [
     "cache",
     "clean"
    ]
   }
  ]
 },
 "repository": {
  "paths": {
   "sources": "/etc/apk/repositories",
   "keys": "/etc/apk/keys"
  },
  "add": {
   "steps": [
    {
     "action": "download",
     "source": "{{ .KeyURL }}",
     "dest": "{{ .KeyPath }}",
     "condition": "{{ .HasKey }}"
    },
    {
     "action": "append",
     "path": "{{ .SourcesPath }}",
     "content": "{{ .URL }}"
    },
    {
     "action": "command",
     "exec": "apk",
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
     "exec": "apk",
     "args": [
      "update"
     ]
    }
   ]
  }
 }
}
