package providers

providers: "mas": {
 "name": "mas",
 "elevated": false,
 "detection": {
  "binary": "mas",
  "files": [
   "/usr/local/bin/mas",
   "/opt/homebrew/bin/mas"
  ],
  "distributions": [
   "darwin"
  ]
 },
 "commands": {
  "install": "install",
  "update": "upgrade",
  "remove": "uninstall",
  "list": "list",
  "search": "search",
  "clean": "reset"
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "mas",
     "args": [
      "signin",
      "--dialog"
     ],
     "condition": "{{ .RequiresAuth }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "mas",
     "args": [
      "signout"
     ]
    }
   ]
  }
 }
}
