package providers

providers: "apt": {
 "name": "apt",
 "elevated": true,
 "detection": {
  "binary": "apt",
  "files": [
   "/etc/apt"
  ],
  "distributions": [
   "debian",
   "ubuntu"
  ]
 },
 "commands": {
  "install": "install -y",
  "update": "update",
  "remove": "remove -y",
  "list": "dpkg --get-selections",
  "search": "search",
  "clean": "clean"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "libssl-dev"
  ],
  "build-essentials": [
   "build-essential",
   "cmake",
   "pkg-config",
   "libfreetype6-dev",
   "libfontconfig1-dev",
   "libxcb-xfixes0-dev",
   "libxkbcommon-dev",
   "python3"
  ]
 },
 "repository": {
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
