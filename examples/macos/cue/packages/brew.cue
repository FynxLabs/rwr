{
  "packages": [
    {
      "names": [
        "git",
        "curl",
        "vim",
        "htop",
        "tree",
        "jq"
      ],
      "action": "install",
      "package_manager": "brew"
    },
    {
      "names": [
        "node",
        "python3",
        "visual-studio-code",
        "docker",
        "iterm2"
      ],
      "profiles": ["dev"],
      "action": "install",
      "package_manager": "brew"
    },
    {
      "names": [
        "firefox",
        "slack",
        "zoom",
        "1password"
      ],
      "profiles": ["work"],
      "action": "install",
      "package_manager": "brew"
    },
    {
      "names": [
        "steam",
        "discord"
      ],
      "profiles": ["gaming"],
      "action": "install",
      "package_manager": "brew"
    }
  ]
}
