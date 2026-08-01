{
  "scripts": [
    {
      "name": "update-system",
      "content": "#!/bin/bash\napt update && apt upgrade -y\napt autoremove -y\n"
    },
    {
      "name": "setup-firewall",
      "content": "#!/bin/bash\nufw --force reset\nufw default deny incoming\nufw default allow outgoing\nufw allow ssh\nufw --force enable\n"
    },
    {
      "name": "setup-dev-env",
      "content": "#!/bin/bash\ncurl -fsSL https://get.docker.com -o get-docker.sh\nsh get-docker.sh\nusermod -aG docker {{.User.username}}\nsystemctl enable docker\n",
      "profiles": ["dev"]
    },
    {
      "name": "configure-server",
      "content": "#!/bin/bash\nsystemctl enable apache2\nsystemctl enable mysql\na2enmod rewrite\nsystemctl restart apache2\n",
      "profiles": ["server"]
    },
    {
      "name": "setup-desktop",
      "content": "#!/bin/bash\ngsettings set org.gnome.desktop.interface gtk-theme 'Adwaita-dark'\ngsettings set org.gnome.shell.extensions.dash-to-dock dock-position BOTTOM\n",
      "profiles": ["desktop"]
    }
  ]
}