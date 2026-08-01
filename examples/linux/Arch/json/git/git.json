{
  "git": [
    {
      "name": "dotfiles",
      "action": "clone",
      "url": "https://github.com/user/dotfiles.git",
      "path": "{{ .User.home }}/.dotfiles",
      "private": false
    },
    {
      "name": "configs",
      "action": "clone",
      "url": "https://github.com/user/configs.git",
      "path": "{{ .User.home }}/configs",
      "private": false
    },
    {
      "name": "work-configs",
      "profiles": [
        "work"
      ],
      "action": "clone",
      "url": "https://github.com/company/work-configs.git",
      "path": "{{ .User.home }}/work/configs",
      "private": true
    },
    {
      "name": "company-tools",
      "profiles": [
        "work"
      ],
      "action": "clone",
      "url": "https://github.com/company/internal-tools.git",
      "path": "{{ .User.home }}/work/tools",
      "private": true
    },
    {
      "name": "awesome-project",
      "profiles": [
        "dev"
      ],
      "action": "clone",
      "url": "https://github.com/user/awesome-project.git",
      "path": "{{ .User.home }}/projects/awesome-project",
      "private": false
    },
    {
      "name": "learning-rust",
      "profiles": [
        "dev"
      ],
      "action": "clone",
      "url": "https://github.com/rust-lang/book.git",
      "path": "{{ .User.home }}/projects/rust-book",
      "private": false
    },
    {
      "name": "rwr",
      "profiles": [
        "dev",
        "work"
      ],
      "action": "clone",
      "url": "https://github.com/FynxLabs/rwr.git",
      "path": "{{ .User.home }}/projects/rwr",
      "private": false
    },
    {
      "name": "gaming-configs",
      "profiles": [
        "gaming"
      ],
      "action": "clone",
      "url": "https://github.com/user/gaming-configs.git",
      "path": "{{ .User.home }}/.config/gaming",
      "private": false
    },
    {
      "name": "game-mods",
      "profiles": [
        "gaming"
      ],
      "action": "clone",
      "url": "https://github.com/user/game-modifications.git",
      "path": "{{ .User.home }}/Games/mods",
      "private": false
    },
    {
      "name": "personal-scripts",
      "profiles": [
        "personal"
      ],
      "action": "clone",
      "url": "https://github.com/user/personal-scripts.git",
      "path": "{{ .User.home }}/scripts",
      "private": true
    },
    {
      "name": "photo-organizer",
      "profiles": [
        "personal"
      ],
      "action": "clone",
      "url": "https://github.com/user/photo-organizer.git",
      "path": "{{ .User.home }}/tools/photo-organizer",
      "private": false
    },
    {
      "name": "security-tools",
      "profiles": [
        "security",
        "work"
      ],
      "action": "clone",
      "url": "https://github.com/security/tools.git",
      "path": "{{ .User.home }}/security/tools",
      "private": true
    },
    {
      "name": "db-migrations",
      "profiles": [
        "database",
        "work",
        "dev"
      ],
      "action": "clone",
      "url": "https://github.com/company/database-migrations.git",
      "path": "{{ .User.home }}/database/migrations",
      "private": true
    }
  ]
}
