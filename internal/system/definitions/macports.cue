package providers

providers: "macports": {
 "name": "macports",
 "elevated": true,
 "detection": {
  "binary": "port",
  "files": [
   "/opt/local/bin/port",
   "/opt/local/etc/macports"
  ],
  "distributions": [
   "darwin"
  ]
 },
 "commands": {
  "install": "install",
  "update": "selfupdate",
  "remove": "uninstall",
  "list": "installed",
  "search": "search",
  "clean": "clean --all all"
 },
 "corePackages": {
  "openssl": [
   "openssl",
   "openssl-devel"
  ],
  "build-essentials": [
   "make",
   "cmake",
   "pkgconfig",
   "freetype",
   "fontconfig"
  ]
 },
 "install": {
  "steps": [
   {
    "action": "download",
    "source": "https://github.com/macports/macports-base/releases/download/v2.8.1/MacPorts-2.8.1-13-Ventura.pkg",
    "dest": "{{ .TempDir }}/MacPorts-2.8.1-13-Ventura.pkg",
    "sha256": "577512628a4b9237b3eccd0e18af28e06855f5d55bd71957c37a9c7c362de5f3"
   },
   {
    "action": "command",
    "exec": "installer",
    "args": [
     "-pkg",
     "{{ .TempDir }}/MacPorts-2.8.1-13-Ventura.pkg",
     "-target",
     "/"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "{{ .TempDir }}/MacPorts-2.8.1-13-Ventura.pkg"
    ]
   },
   {
    "action": "command",
    "exec": "port",
    "args": [
     "selfupdate"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "port",
    "args": [
     "-f",
     "uninstall",
     "installed"
    ]
   },
   {
    "action": "command",
    "exec": "rm",
    "args": [
     "-rf",
     "/opt/local",
     "/Applications/DarwinPorts",
     "/Applications/MacPorts",
     "/Library/LaunchDaemons/org.macports.*",
     "/Library/Receipts/DarwinPorts*.pkg",
     "/Library/Receipts/MacPorts*.pkg",
     "/Library/StartupItems/DarwinPortsStartup",
     "/Library/Tcl/darwinports1.0",
     "/Library/Tcl/macports1.0",
     "~/.macports"
    ]
   }
  ]
 },
 "repository": {
  "paths": {
   "sources": "/opt/local/etc/macports/sources.conf"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "port",
     "args": [
      "sync"
     ]
    },
    {
     "action": "append",
     "path": "{{ .SourcesPath }}",
     "content": "{{ .URL }}"
    },
    {
     "action": "command",
     "exec": "port",
     "args": [
      "selfupdate"
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
     "action": "command",
     "exec": "port",
     "args": [
      "selfupdate"
     ]
    }
   ]
  }
 }
}
