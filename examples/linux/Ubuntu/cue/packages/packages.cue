{
  "packages": [
    {
      "names": [
        "git",
        "curl",
        "vim",
        "htop",
        "tree",
        "unzip",
        "build-essential"
      ],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "nodejs",
        "npm",
        "python3",
        "python3-pip",
        "code"
      ],
      "profiles": ["dev"],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "firefox",
        "libreoffice",
        "keepassxc"
      ],
      "profiles": ["work"],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "steam",
        "wine",
        "discord"
      ],
      "profiles": ["gaming"],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "docker.io",
        "docker-compose"
      ],
      "profiles": ["docker"],
      "action": "install",
      "package_manager": "apt"
    },
    {
      "names": [
        "postgresql",
        "postgresql-contrib"
      ],
      "profiles": ["database"],
      "action": "install",
      "package_manager": "apt"
    }
  ]
}