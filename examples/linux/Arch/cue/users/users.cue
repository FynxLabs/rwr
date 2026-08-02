{
  "users": [
    {
      "name": "{{ .User.username }}",
      "action": "modify",
      "add_groups": [
        "wheel",
        "users"
      ]
    },
    {
      "name": "{{ .User.username }}",
      "profiles": [
        "dev"
      ],
      "action": "modify",
      "add_groups": [
        "docker",
        "developers",
        "git"
      ]
    },
    {
      "name": "{{ .User.username }}",
      "profiles": [
        "work"
      ],
      "action": "modify",
      "add_groups": [
        "docker",
        "sudo",
        "developers"
      ]
    },
    {
      "name": "{{ .User.username }}",
      "profiles": [
        "gaming"
      ],
      "action": "modify",
      "add_groups": [
        "games",
        "audio",
        "video"
      ]
    },
    {
      "name": "developer",
      "profiles": [
        "dev",
        "work"
      ],
      "action": "create",
      "shell": "/bin/zsh",
      "home": "/home/developer",
      "groups": [
        "developers",
        "docker"
      ]
    },
    {
      "name": "gamer",
      "profiles": [
        "gaming"
      ],
      "action": "create",
      "shell": "/bin/bash",
      "home": "/home/gamer",
      "groups": [
        "games",
        "audio",
        "video"
      ]
    }
  ],
  "groups": [
    {
      "name": "users",
      "action": "create"
    },
    {
      "name": "developers",
      "profiles": [
        "dev",
        "work"
      ],
      "action": "create"
    },
    {
      "name": "docker",
      "profiles": [
        "dev",
        "work"
      ],
      "action": "create"
    },
    {
      "name": "games",
      "profiles": [
        "gaming"
      ],
      "action": "create"
    },
    {
      "name": "security",
      "profiles": [
        "security",
        "work"
      ],
      "action": "create"
    },
    {
      "name": "database",
      "profiles": [
        "database",
        "dev"
      ],
      "action": "create"
    }
  ]
}
