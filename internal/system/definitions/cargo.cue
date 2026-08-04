package providers

providers: "cargo": {
 "name": "cargo",
 "elevated": false,
 "detection": {
  "binary": "cargo",
  "files": [
   "~/.cargo/bin/cargo",
   "~/.rustup"
  ],
  "distributions": [
   "linux",
   "darwin",
   "windows"
  ]
 },
 "commands": {
  "install": "install",
  "update": "install-update --all",
  "remove": "uninstall",
  "list": "install --list",
  "listExplicit": "install --list",
  "search": "search",
  "clean": "cache --autoclean"
 },
 "corePackages": {
  "openssl": [],
  "build-essentials": []
 },
 "repository": {
  "paths": {
   "config": "~/.cargo/config.toml"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "cargo",
     "args": [
      "update"
     ]
    },
    {
     "action": "write",
     "dest": "{{ .ConfigPath }}",
     "content": "[registries.{{ .Name }}]\nindex = \"{{ .URL }}\"\n",
     "condition": "{{ .IsCustomRegistry }}"
    },
    {
     "action": "command",
     "exec": "cargo",
     "args": [
      "login",
      "--registry",
      "{{ .Name }}",
      "{{ .Token }}"
     ],
     "condition": "{{ .HasToken }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "cargo",
     "args": [
      "logout",
      "--registry",
      "{{ .Name }}"
     ],
     "condition": "{{ .HasToken }}"
    },
    {
     "action": "remove_section",
     "path": "{{ .ConfigPath }}",
     "section": "registries.{{ .Name }}",
     "condition": "{{ .IsCustomRegistry }}"
    }
   ]
  }
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "curl",
    "args": [
     "--proto",
     "=https",
     "--tlsv1.2",
     "-sSf",
     "https://sh.rustup.rs",
     "-o",
     "{{ .TempDir }}/rustup-init.sh"
    ]
   },
   {
    "action": "command",
    "exec": "sh",
    "args": [
     "{{ .TempDir }}/rustup-init.sh",
     "-y"
    ]
   },
   {
    "action": "command",
    "exec": "cargo",
    "args": [
     "install",
     "cargo-update",
     "--features",
     "vendored-openssl"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "rustup",
    "args": [
     "self",
     "uninstall",
     "-y"
    ]
   }
  ]
 }
}
