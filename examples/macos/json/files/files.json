{
  "files": [
    {
      "name": ".bashrc",
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "# Custom .bashrc content\nalias ll='ls -alF'\nexport PATH=$PATH:$HOME/.local/bin\n"
    },
    {
      "name": ".gitignore",
      "action": "copy",
      "target": "{{ .User.home }}/",
      "source": "./src/"
    }
  ],
  "directories": [
    {
      "name": ".config",
      "action": "copy",
      "source": "./src/",
      "target": "{{ .User.home }}/"
    }
  ],
  "templates": [
    {
      "name": ".profile",
      "action": "copy",
      "source": "./src/",
      "target": "{{ .User.home }}/"
    }
  ]
}