package providers

providers: "scoop": {
 "name": "scoop",
 "elevated": false,
 "detection": {
  "binary": "scoop",
  "files": [
   "%USERPROFILE%\\scoop\\shims\\scoop.cmd"
  ],
  "distributions": [
   "windows"
  ]
 },
 "commands": {
  "install": "install",
  "update": "update",
  "remove": "uninstall",
  "list": "list",
  "search": "search",
  "clean": "cache rm *"
 },
 "corePackages": {
  "openssl": [
   "openssl"
  ],
  "build-essentials": [
   "make",
   "cmake",
   "freetype",
   "fontconfig"
  ]
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force"
    ]
   },
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "iex (New-Object System.Net.WebClient).DownloadString('https://get.scoop.sh')"
    ]
   },
   {
    "action": "command",
    "exec": "scoop",
    "args": [
     "bucket",
     "add",
     "main"
    ]
   },
   {
    "action": "command",
    "exec": "scoop",
    "args": [
     "bucket",
     "add",
     "extras"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "scoop",
    "args": [
     "uninstall",
     "scoop",
     "-p"
    ]
   }
  ]
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "scoop",
     "args": [
      "bucket",
      "add",
      "{{ .Name }}",
      "{{ .URL }}"
     ]
    },
    {
     "action": "command",
     "exec": "scoop",
     "args": [
      "bucket",
      "add",
      "{{ .Name }}"
     ],
     "condition": "{{ not .URL }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "scoop",
     "args": [
      "bucket",
      "rm",
      "{{ .Name }}"
     ]
    }
   ]
  }
 }
}
