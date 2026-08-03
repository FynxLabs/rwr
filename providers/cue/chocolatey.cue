package providers

providers: "chocolatey": {
 "name": "chocolatey",
 "elevated": true,
 "detection": {
  "binary": "choco",
  "files": [
   "%ProgramData%\\chocolatey\\bin\\choco.exe"
  ],
  "distributions": [
   "windows"
  ]
 },
 "commands": {
  "install": "install -y",
  "update": "upgrade -y all",
  "remove": "uninstall -y",
  "list": "list --local-only",
  "search": "search",
  "clean": "cache delete"
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))"
    ]
   }
  ]
 },
 "remove": {
  "steps": [
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "choco uninstall chocolatey -y"
    ]
   }
  ]
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
 "repository": {
  "paths": {
   "sources": "%ProgramData%\\chocolatey\\config"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "choco",
     "args": [
      "source",
      "add",
      "--name={{ .Name }}",
      "--source={{ .URL }}"
     ]
    },
    {
     "action": "command",
     "exec": "choco",
     "args": [
      "source",
      "add",
      "--name={{ .Name }}",
      "--source={{ .URL }}",
      "--user={{ .Username }}",
      "--password={{ .Password }}"
     ],
     "condition": "{{ .HasAuthentication }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "choco",
     "args": [
      "source",
      "remove",
      "--name={{ .Name }}"
     ]
    }
   ]
  }
 }
}
