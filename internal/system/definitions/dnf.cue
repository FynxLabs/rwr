package providers

providers: "dnf": {
 "name": "dnf",
 "elevated": true,
 "detection": {
  "binary": "dnf",
  "files": [
   "/etc/dnf/dnf.conf",
   "/var/lib/dnf"
  ],
  "distributions": [
   "fedora",
   "rhel",
   "openmandriva"
  ]
 },
 "commands": {
  "install": "install -y",
  "update": "update -y",
  "remove": "remove -y",
  "list": "list installed",
  "listExplicit": "repoquery --userinstalled --qf %{name}",
  "search": "search",
  "clean": "clean all"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "openssl-devel"
  ],
  "build-essentials": [
   "make",
   "cmake",
   "freetype-devel",
   "fontconfig-devel",
   "libxcb-devel",
   "libxkbcommon-devel",
   "g++"
  ]
 },
 "repository": {
  "paths": {
   "sources": "/etc/yum.repos.d",
   "keys": "/etc/pki/rpm-gpg"
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
     "exec": "rpm",
     "args": [
      "--import",
      "{{ .TempKeyPath }}"
     ]
    },
    {
     "action": "copy",
     "source": "{{ .TempKeyPath }}",
     "dest": "{{ .KeyPath }}"
    },
    {
     "action": "write",
     "dest": "{{ .SourcesPath }}/{{ .Name }}.repo",
     "content": "[{{ .Name }}]\nname={{ .Description }}\nbaseurl={{ .URL }}\nenabled=1\ngpgcheck=1\ngpgkey={{ .KeyPath }}\n"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "remove",
     "path": "{{ .SourcesPath }}/{{ .Name }}.repo"
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
 },
 "alternatives": {
  "openmandriva": {
   "corePackages": {
    "openssl": [
     "openssl",
     "lib64openssl-devel"
    ],
    "build-essentials": [
     "make",
     "cmake",
     "lib64freetype6-devel",
     "lib64fontconfig-devel",
     "lib64xcb-devel",
     "lib64xkbcommon-devel",
     "gcc-c++"
    ]
   }
  }
 }
}
