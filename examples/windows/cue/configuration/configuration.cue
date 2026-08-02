{
  "configurations": [
    {
      "name": "show-file-extensions",
      "action": "set",
      "tool": "windows_registry",
      "elevated": true,
      "path": "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\Advanced",
      "key": "HideFileExt",
      "type": "dword",
      "value": 0
    }
  ]
}
