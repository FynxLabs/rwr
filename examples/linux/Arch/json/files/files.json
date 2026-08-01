{
  "files": [
    {
      "name": ".bashrc",
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "# Custom .bashrc content\nalias ll='ls -alF'\nalias la='ls -A'\nalias l='ls -CF'\nexport PATH=$PATH:$HOME/.local/bin\nexport EDITOR=nvim\n"
    },
    {
      "name": ".vimrc",
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "\" Basic vim configuration\nset number\nset relativenumber\nset tabstop=4\nset shiftwidth=4\nset expandtab\nsyntax on\n"
    },
    {
      "name": ".gitconfig-work",
      "profiles": [
        "work"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "[user]\n    name = Work User\n    email = user@company.com\n[core]\n    editor = code --wait\n[push]\n    default = simple\n"
    },
    {
      "name": "work-ssh-config",
      "profiles": [
        "work"
      ],
      "action": "copy",
      "source": "./src/ssh/work_config",
      "target": "{{ .User.home }}/.ssh/config",
      "mode": 384
    },
    {
      "name": ".gitconfig-dev",
      "profiles": [
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "[user]\n    name = Developer\n    email = dev@personal.com\n[core]\n    editor = nvim\n[alias]\n    st = status\n    co = checkout\n    br = branch\n"
    },
    {
      "name": ".zshrc",
      "profiles": [
        "dev",
        "work"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "# Zsh configuration\nexport ZSH=\"$HOME/.oh-my-zsh\"\nZSH_THEME=\"robbyrussell\"\nplugins=(git docker kubectl)\nsource $ZSH/oh-my-zsh.sh\n"
    },
    {
      "name": "gamemode.ini",
      "profiles": [
        "gaming"
      ],
      "action": "copy",
      "source": "./src/gamemode.ini",
      "target": "{{ .User.home }}/.config/gamemode/"
    },
    {
      "name": ".aliases",
      "profiles": [
        "personal"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "content": "# Personal aliases\nalias music='vlc ~/Music'\nalias photos='gimp'\nalias backup='rsync -av ~/Documents/ ~/Backup/'\n"
    }
  ],
  "directories": [
    {
      "name": ".config",
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": ".local/bin",
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": "work",
      "profiles": [
        "work"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": ".ssh",
      "profiles": [
        "work",
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 448
    },
    {
      "name": "projects",
      "profiles": [
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": ".config/nvim",
      "profiles": [
        "dev"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": ".config/gamemode",
      "profiles": [
        "gaming"
      ],
      "action": "create",
      "target": "{{ .User.home }}/",
      "mode": 493
    },
    {
      "name": "Games",
      "profiles": [
        "gaming",
        "personal"
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
      "source": "./src/.profile",
      "target": "{{ .User.home }}/"
    },
    {
      "name": "work-environment",
      "profiles": [
        "work"
      ],
      "action": "copy",
      "source": "./src/work-environment.j2",
      "target": "{{ .User.home }}/.work-env",
      "variables": {
        "company": "{{ .UserDefined.company }}",
        "department": "{{ .UserDefined.department }}"
      }
    },
    {
      "name": "nvim-config",
      "profiles": [
        "dev"
      ],
      "action": "copy",
      "source": "./src/nvim/init.lua.j2",
      "target": "{{ .User.home }}/.config/nvim/init.lua",
      "variables": {
        "theme": "{{ .UserDefined.editor_theme }}",
        "plugins": "{{ .UserDefined.editor_plugins }}"
      }
    },
    {
      "name": "personal-scripts",
      "profiles": [
        "personal"
      ],
      "action": "copy",
      "source": "./src/personal-scripts.j2",
      "target": "{{ .User.home }}/.local/bin/personal-tools",
      "mode": 493
    }
  ]
}
