package providers

providers: "nix": {
 "name": "nix",
 "elevated": false,
 "detection": {
  "binary": "nix-env",
  "files": [
   "/nix/store",
   "~/.nix-profile",
   "/etc/nix",
   "/etc/nixos",
   "~/.config/nixpkgs"
  ],
  "distributions": [
   "linux",
   "darwin"
  ]
 },
 "commands": {
  "install": "-i",
  "update": "-u '*'",
  "remove": "-e",
  "list": "-q",
  "search": "nix search",
  "clean": "nix-collect-garbage -d"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "openssl.dev"
  ],
  "build-essentials": [
   "gnumake",
   "cmake",
   "freetype",
   "fontconfig",
   "pkg-config",
   "libxcb",
   "libxkbcommon",
   "python3"
  ]
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "curl",
    "args": [
     "-L",
     "https://nixos.org/nix/install",
     "--output",
     "{{ .TempDir }}/nix-install"
    ]
   },
   {
    "action": "command",
    "exec": "chmod",
    "args": [
     "+x",
     "{{ .TempDir }}/nix-install"
    ]
   },
   {
    "action": "command",
    "exec": "sh",
    "args": [
     "{{ .TempDir }}/nix-install"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "{{ .TempDir }}/nix-install"
    ]
   },
   {
    "action": "command",
    "exec": "nix-channel",
    "args": [
     "--add",
     "https://nixos.org/channels/nixpkgs-unstable",
     "nixpkgs"
    ]
   },
   {
    "action": "command",
    "exec": "nix-channel",
    "args": [
     "--update"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "curl",
    "args": [
     "-L",
     "https://nixos.org/nix/uninstall",
     "--output",
     "{{ .TempDir }}/nix-uninstall"
    ]
   },
   {
    "action": "command",
    "exec": "chmod",
    "args": [
     "+x",
     "{{ .TempDir }}/nix-uninstall"
    ]
   },
   {
    "action": "command",
    "exec": "sh",
    "args": [
     "{{ .TempDir }}/nix-uninstall"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "{{ .TempDir }}/nix-uninstall"
    ]
   }
  ]
 },
 "repository": {
  "paths": {
   "config": "/etc/nix/nix.conf"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "nix-channel",
     "args": [
      "--add",
      "{{ .URL }}",
      "{{ .Name }}"
     ]
    },
    {
     "action": "command",
     "exec": "nix-channel",
     "args": [
      "--update",
      "{{ .Name }}"
     ]
    },
    {
     "action": "write",
     "dest": "{{ .ConfigPath }}/config.nix",
     "content": "{\n  packageOverrides = pkgs: {\n    {{ .Name }} = import (builtins.fetchTarball {\n      url = \"{{ .URL }}\";\n      sha256 = \"{{ .SHA256 }}\";\n    }) {\n      inherit pkgs;\n    };\n  };\n}\n",
     "condition": "{{ .IsOverlay }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "nix-channel",
     "args": [
      "--remove",
      "{{ .Name }}"
     ]
    },
    {
     "action": "remove",
     "path": "{{ .ConfigPath }}/config.nix",
     "condition": "{{ .IsOverlay }}"
    }
   ]
  }
 }
}
