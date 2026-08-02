package providers

providers: "winget": {
 "name": "winget",
 "elevated": false,
 "detection": {
  "binary": "winget",
  "files": [
   "%LOCALAPPDATA%\\Microsoft\\WindowsApps\\winget.exe"
  ],
  "distributions": [
   "windows"
  ]
 },
 "commands": {
  "install": "install --silent",
  "update": "upgrade --all",
  "remove": "uninstall",
  "list": "list",
  "search": "search",
  "clean": "source reset"
 },
 "corePackages": {
  "openssl": [
   "OpenSSL.OpenSSL"
  ],
  "build-essentials": [
   "GnuWin32.Make",
   "Kitware.CMake",
   "FreeType.FreeType",
   "FontConfig.FontConfig"
  ]
 },
 "install": {
  "steps": [
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "Invoke-WebRequest -Uri https://github.com/microsoft/winget-cli/releases/download/v1.5.9371.0/Microsoft.DesktopAppInstaller_8wekyb3d8bbwe.appxbundle -OutFile winget.appxbundle"
    ]
   },
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "Add-AppxPackage -Path winget.appxbundle"
    ]
   },
   {
    "action": "command",
    "exec": "powershell",
    "args": [
     "-Command",
     "Remove-Item winget.appxbundle"
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
     "Get-AppxPackage Microsoft.DesktopAppInstaller | Remove-AppxPackage"
    ]
   }
  ]
 },
 "repository": {
  "paths": {
   "sources": "%LOCALAPPDATA%\\Microsoft\\WindowsApps\\Sources"
  },
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "winget",
     "args": [
      "source",
      "add",
      "-n",
      "{{ .Name }}",
      "--url",
      "{{ .URL }}"
     ]
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "winget",
     "args": [
      "source",
      "remove",
      "-n",
      "{{ .Name }}"
     ]
    }
   ]
  }
 }
}
