#!/usr/bin/env bash

# Check bash version
bash_version=$(bash --version | grep '^GNU bash' | awk '{print $4}' | cut -d '.' -f1)
if (( bash_version >= 4 )); then
  echo "Bash version 4.x or higher detected, skipping update..."
  exit 0
fi

# Import package manager details
# shellcheck source=/dev/null
source /tmp/os.env

# Update/install Bash
if [[ "${OS}" == "macos" ]]; then
  echo ">>> Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  echo ">>> Updating Bash via Homebrew..."
  ${PKG_INSTALL} bash >/dev/null 2>&1
else
  echo ">>> Updating Bash via local package manager..."
  if [[ "${PKG}" == "apt" ]]; then
    # Add Debian backports repository for updated Bash version if necessary
    echo "deb http://ftp.debian.org/debian buster-backports main" | sudo tee -a /etc/apt/sources.list.d/buster-backports.list
    sudo apt update >/dev/null 2>&1
  fi
  ${PKG_INSTALL} bash >/dev/null 2>&1
fi

echo "Bash has been updated to version $(bash --version | grep '^GNU bash' | awk '{print $4}')"
