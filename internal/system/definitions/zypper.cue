package providers

providers: "zypper": {
 "name": "zypper",
 "elevated": true,
 "detection": {
  "binary": "zypper",
  "files": [
   "/etc/zypp",
   "/var/lib/zypp"
  ],
  "distributions": [
   "opensuse",
   "suse"
  ]
 },
 "commands": {
  "install": "install -y",
  "update": "update -y",
  "remove": "remove -y",
  "list": "packages --installed-only",
  "search": "search",
  "clean": "clean"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "libopenssl-devel"
  ],
  "build-essentials": [
   "make",
   "cmake",
   "freetype-devel",
   "fontconfig-devel",
   "libxcb-devel",
   "libxkbcommon-devel"
  ]
 },
 "repository": {
  "paths": {
   "sources": "/etc/zypp/repos.d",
   "keys": "/etc/pki/rpm-gpg"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "zypper",
     "args": [
      "addrepo",
      "{{ .URL }}",
      "{{ .Name }}"
     ]
    },
    {
     "action": "download",
     "source": "{{ .KeyURL }}",
     "dest": "{{ .KeyPath }}"
    },
    {
     "action": "command",
     "exec": "rpm",
     "args": [
      "--import",
      "{{ .KeyPath }}"
     ]
    },
    {
     "action": "command",
     "exec": "zypper",
     "args": [
      "refresh"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "zypper",
     "args": [
      "removerepo",
      "{{ .Name }}"
     ]
    },
    {
     "action": "command",
     "exec": "rpm",
     "args": [
      "--erase",
      "gpg-pubkey-{{ .KeyID }}"
     ]
    }
   ]
  }
 }
}
