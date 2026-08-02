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
    },
    {
      "name": ".vimrc",
      "profiles": [
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "set number\nset tabstop=4\nset shiftwidth=4\nset expandtab\n"
    }
  ],
  "directories": [
    {
      "name": "Projects",
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": ".config/code",
      "profiles": [
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
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
