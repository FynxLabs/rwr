package providers

providers: "snap": {
 "name": "snap",
 "elevated": true,
 "detection": {
  "binary": "snap",
  "files": [
   "/usr/bin/snap",
   "/var/lib/snapd",
   "/snap"
  ],
  "distributions": [
   "linux"
  ]
 },
 "commands": {
  "install": "install",
  "update": "refresh",
  "remove": "remove",
  "list": "list",
  "search": "find",
  "clean": "refresh"
 },
 "repository": {
  "add": {
   "steps": [
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "set",
      "system",
      "proxy.http={{ .ProxyURL }}"
     ],
     "condition": "{{ .HasProxy }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "set",
      "system",
      "proxy.https={{ .ProxyURL }}"
     ],
     "condition": "{{ .HasProxy }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "install",
      "{{ .Name }}"
     ],
     "condition": "{{ .IsSnapStore }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "install",
      "{{ .Path }}",
      "--dangerous"
     ],
     "condition": "{{ .IsLocalSnap }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "connect",
      "{{ .Name }}:{{ .Interface }}",
      "{{ .Slot }}"
     ],
     "condition": "{{ and .HasInterfaces .HasSlot }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "connect",
      "{{ .Name }}:{{ .Interface }}"
     ],
     "condition": "{{ and .HasInterfaces (not .HasSlot) }}"
    }
   ]
  },
  "remove": {
   "steps": [
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "remove",
      "{{ .Name }}"
     ]
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "unset",
      "system",
      "proxy.http"
     ],
     "condition": "{{ .HasProxy }}"
    },
    {
     "action": "command",
     "exec": "snap",
     "args": [
      "unset",
      "system",
      "proxy.https"
     ],
     "condition": "{{ .HasProxy }}"
    }
   ]
  }
 }
}
